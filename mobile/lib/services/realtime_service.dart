import 'dart:async';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'api.dart';
import 'push_service.dart';
import 'session.dart';

/// Lightweight WSS client: connects to /v1/stream and refreshes alarm badge on any frame.
/// Full Protobuf decode can be layered later; binary frames still trigger REST snapshot.
class RealtimeService {
  RealtimeService._();
  static final RealtimeService instance = RealtimeService._();

  WebSocketChannel? _ch;
  StreamSubscription? _sub;
  Timer? _reconnect;
  int _alarmBadge = 0;
  final _ctrl = StreamController<int>.broadcast();

  Stream<int> get alarmBadgeStream => _ctrl.stream;
  int get alarmBadge => _alarmBadge;

  Future<void> start() async {
    await stop();
    final token = await Session.accessToken();
    if (token == null || token.isEmpty) return;
    final httpBase = apiBase.replaceAll('/api/v1', '');
    final wsBase = httpBase.replaceFirst('http://', 'ws://').replaceFirst('https://', 'wss://');
    final uri = Uri.parse('$wsBase/v1/stream?token=${Uri.encodeComponent(token)}');
    try {
      _ch = WebSocketChannel.connect(uri);
      _sub = _ch!.stream.listen(
        (_) {
          unawaited(_refreshBadge());
        },
        onError: (_) => _scheduleReconnect(),
        onDone: _scheduleReconnect,
      );
      await _refreshBadge();
    } catch (_) {
      _scheduleReconnect();
    }
  }

  Future<void> _refreshBadge() async {
    try {
      final data = await Api.get('/alarms?status=raised&page=1&pageSize=1');
      final total = (data is Map) ? (data['total'] as num?)?.toInt() ?? 0 : 0;
      if (total > _alarmBadge && _alarmBadge >= 0) {
        unawaited(PushService.instance.showSummary(
          title: '新告警',
          body: '未确认告警 $total 条',
        ));
      }
      _alarmBadge = total;
      if (!_ctrl.isClosed) _ctrl.add(total);
    } catch (_) {}
  }

  void _scheduleReconnect() {
    _reconnect?.cancel();
    _reconnect = Timer(const Duration(seconds: 5), () => start());
  }

  Future<void> stop() async {
    _reconnect?.cancel();
    await _sub?.cancel();
    await _ch?.sink.close();
    _sub = null;
    _ch = null;
  }
}
