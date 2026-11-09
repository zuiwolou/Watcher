import 'dart:async';
import 'dart:convert';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';

import 'api.dart';

enum OutboxStatus { pending, syncing, failed, done }

class OutboxItem {
  OutboxItem({
    required this.id,
    required this.method,
    required this.path,
    required this.body,
    required this.createdAt,
    this.status = OutboxStatus.pending,
    this.attempts = 0,
    this.lastError = '',
    this.label = '',
  });

  final String id;
  final String method;
  final String path;
  final Map<String, dynamic> body;
  final DateTime createdAt;
  OutboxStatus status;
  int attempts;
  String lastError;
  String label;

  Map<String, dynamic> toJson() => {
        'id': id,
        'method': method,
        'path': path,
        'body': body,
        'createdAt': createdAt.toIso8601String(),
        'status': status.name,
        'attempts': attempts,
        'lastError': lastError,
        'label': label,
      };

  static OutboxItem fromJson(Map<String, dynamic> j) => OutboxItem(
        id: '${j['id']}',
        method: '${j['method'] ?? 'POST'}',
        path: '${j['path']}',
        body: Map<String, dynamic>.from(j['body'] as Map? ?? {}),
        createdAt: DateTime.tryParse('${j['createdAt']}') ?? DateTime.now(),
        status: OutboxStatus.values.firstWhere(
          (e) => e.name == j['status'],
          orElse: () => OutboxStatus.pending,
        ),
        attempts: (j['attempts'] as num?)?.toInt() ?? 0,
        lastError: '${j['lastError'] ?? ''}',
        label: '${j['label'] ?? ''}',
      );
}

/// Local offline Outbox: queue exec POSTs and replay when online.
class OutboxService {
  OutboxService._();
  static final OutboxService instance = OutboxService._();

  static const _key = 'watcher_outbox_v1';
  final _uuid = const Uuid();
  final _ctrl = StreamController<List<OutboxItem>>.broadcast();
  Timer? _timer;
  bool _flushing = false;

  Stream<List<OutboxItem>> get changes => _ctrl.stream;

  Future<List<OutboxItem>> list() async {
    final sp = await SharedPreferences.getInstance();
    final raw = sp.getString(_key);
    if (raw == null || raw.isEmpty) return [];
    final arr = jsonDecode(raw) as List;
    return arr.map((e) => OutboxItem.fromJson(Map<String, dynamic>.from(e as Map))).toList();
  }

  Future<void> _save(List<OutboxItem> items) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setString(_key, jsonEncode(items.map((e) => e.toJson()).toList()));
    _ctrl.add(items);
  }

  Future<OutboxItem> enqueue({
    required String path,
    required Map<String, dynamic> body,
    String label = '',
  }) async {
    // ensure idempotency key
    body.putIfAbsent('clientRequestId', () => _uuid.v4());
    final items = await list();
    final item = OutboxItem(
      id: _uuid.v4(),
      method: 'POST',
      path: path,
      body: body,
      createdAt: DateTime.now(),
      label: label,
    );
    items.insert(0, item);
    await _save(items);
    unawaited(flush());
    return item;
  }

  /// Online submit; on network/transient failure → enqueue.
  /// Returns `{queued:false, data:...}` or `{queued:true, itemId:...}`.
  Future<Map<String, dynamic>> submitOrEnqueue({
    required String path,
    required Map<String, dynamic> body,
    String label = '',
  }) async {
    body.putIfAbsent('clientRequestId', () => _uuid.v4());
    try {
      final data = await Api.post(path, body);
      return {'queued': false, 'data': data};
    } on ApiException catch (e) {
      if (e.code.contains('STATE_INVALID') || e.code.contains('VALIDATION') || e.code.contains('AUTH')) {
        rethrow;
      }
      final item = await enqueue(path: path, body: body, label: label);
      return {'queued': true, 'itemId': item.id, 'error': e.toString()};
    } catch (e) {
      final item = await enqueue(path: path, body: body, label: label);
      return {'queued': true, 'itemId': item.id, 'error': e.toString()};
    }
  }

  void startAutoFlush() {
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 12), (_) => flush());
    Connectivity().onConnectivityChanged.listen((_) => flush());
    unawaited(flush());
  }

  void stop() {
    _timer?.cancel();
    _timer = null;
  }

  Future<void> flush() async {
    if (_flushing) return;
    _flushing = true;
    try {
      final conn = await Connectivity().checkConnectivity();
      final offline = conn.every((c) => c == ConnectivityResult.none);
      if (offline) return;

      final items = await list();
      var changed = false;
      for (final item in items.where((e) => e.status == OutboxStatus.pending || e.status == OutboxStatus.failed)) {
        if (item.attempts >= 8) {
          item.status = OutboxStatus.failed;
          item.lastError = item.lastError.isEmpty ? 'max attempts' : item.lastError;
          changed = true;
          continue;
        }
        item.status = OutboxStatus.syncing;
        item.attempts += 1;
        changed = true;
        try {
          await Api.post(item.path, item.body);
          item.status = OutboxStatus.done;
          item.lastError = '';
        } on ApiException catch (e) {
          if (e.code.contains('STATE_INVALID') || e.code.contains('VALIDATION')) {
            item.status = OutboxStatus.failed;
            item.lastError = e.toString();
          } else {
            item.status = OutboxStatus.pending;
            item.lastError = e.toString();
          }
        } catch (e) {
          item.status = OutboxStatus.pending;
          item.lastError = e.toString();
        }
      }
      // prune done
      final next = items.where((e) => e.status != OutboxStatus.done).toList();
      if (changed || next.length != items.length) await _save(next);
    } finally {
      _flushing = false;
    }
  }

  Future<void> retry(String id) async {
    final items = await list();
    final hit = items.where((e) => e.id == id).firstOrNull;
    if (hit == null) return;
    hit.status = OutboxStatus.pending;
    hit.lastError = '';
    await _save(items);
    await flush();
  }

  Future<void> remove(String id) async {
    final items = await list();
    items.removeWhere((e) => e.id == id);
    await _save(items);
  }
}

extension _FirstOrNull<E> on Iterable<E> {
  E? get firstOrNull {
    final it = iterator;
    if (it.moveNext()) return it.current;
    return null;
  }
}
