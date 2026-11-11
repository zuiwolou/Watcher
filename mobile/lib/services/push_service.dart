import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';

import 'api.dart';

/// Push skeleton: local notifications + device token registry.
/// Wire FCM/APNs later by calling [onRemoteMessage] from the vendor SDK.
class PushService {
  PushService._();
  static final PushService instance = PushService._();

  final _plugin = FlutterLocalNotificationsPlugin();
  final _uuid = const Uuid();
  Timer? _poll;
  String? deviceToken;
  int _lastRaised = -1;

  Future<void> init() async {
    const android = AndroidInitializationSettings('@mipmap/ic_launcher');
    const ios = DarwinInitializationSettings();
    await _plugin.initialize(
      const InitializationSettings(android: android, iOS: ios),
      onDidReceiveNotificationResponse: (resp) {
        // Payload is alarm summary; UI should open Alarms tab via app lifecycle hooks.
        // Stored for debugging / future deep-link router.
        // ignore: avoid_print
        print('notification tap: ${resp.payload}');
      },
    );
    if (Platform.isAndroid) {
      await _plugin
          .resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>()
          ?.createNotificationChannel(const AndroidNotificationChannel(
            'watcher_alarms',
            '告警通知',
            description: '危废管控告警摘要',
            importance: Importance.high,
          ));
    }
    await Permission.notification.request();
    deviceToken = await _ensureDeviceToken();
  }

  Future<String> _ensureDeviceToken() async {
    final sp = await SharedPreferences.getInstance();
    var t = sp.getString('pushDeviceToken');
    if (t == null || t.isEmpty) {
      // Placeholder until FCM/APNs token is injected.
      t = 'local-${Platform.operatingSystem}-${_uuid.v4()}';
      await sp.setString('pushDeviceToken', t);
    }
    return t;
  }

  /// Vendor SDK should call this with the real token (FCM/APNs).
  Future<void> registerVendorToken(String token) async {
    deviceToken = token;
    final sp = await SharedPreferences.getInstance();
    await sp.setString('pushDeviceToken', token);
  }

  Future<void> showSummary({
    required String title,
    required String body,
    String? payload,
  }) async {
    await _plugin.show(
      DateTime.now().millisecondsSinceEpoch ~/ 1000,
      title,
      body,
      const NotificationDetails(
        android: AndroidNotificationDetails(
          'watcher_alarms',
          '告警通知',
          channelDescription: '危废管控告警摘要',
          importance: Importance.high,
          priority: Priority.high,
        ),
        iOS: DarwinNotificationDetails(),
      ),
      payload: payload,
    );
  }

  void onRemoteMessage(Map<String, dynamic> data) {
    final title = '${data['title'] ?? data['level'] ?? '告警'}';
    final body = '${data['body'] ?? data['message'] ?? jsonEncode(data)}';
    unawaited(showSummary(title: title, body: body, payload: jsonEncode(data)));
  }

  /// Foreground poll as MVP stand-in for vendor push delivery.
  void startAlarmPolling() {
    _poll?.cancel();
    _poll = Timer.periodic(const Duration(seconds: 45), (_) => _pollAlarms());
    unawaited(_pollAlarms());
  }

  void stop() {
    _poll?.cancel();
    _poll = null;
  }

  Future<void> _pollAlarms() async {
    try {
      final data = await Api.get('/alarms?status=raised&page=1&pageSize=1');
      final total = (data is Map) ? (data['total'] as num?)?.toInt() ?? 0 : 0;
      if (_lastRaised >= 0 && total > _lastRaised) {
        await showSummary(
          title: '新告警',
          body: '未确认告警 $total 条，点击查看详情',
          payload: jsonEncode({'type': 'alarm_badge', 'total': total}),
        );
      }
      _lastRaised = total;
    } catch (_) {
      // ignore offline
    }
  }
}
