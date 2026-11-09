import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

class Session {
  static Future<SharedPreferences> get _sp => SharedPreferences.getInstance();

  static Future<String?> accessToken() async => (await _sp).getString('accessToken');
  static Future<String?> refreshToken() async => (await _sp).getString('refreshToken');

  static Future<List<String>> permissions() async {
    final raw = (await _sp).getString('permissions') ?? '[]';
    return (jsonDecode(raw) as List).map((e) => '$e').toList();
  }

  static Future<bool> has(String p) async => (await permissions()).contains(p);

  static Future<bool> mustChangePassword() async =>
      (await _sp).getBool('mustChangePassword') ?? false;

  static Future<Map<String, dynamic>> user() async {
    final raw = (await _sp).getString('user') ?? '{}';
    return Map<String, dynamic>.from(jsonDecode(raw) as Map);
  }

  static Future<void> saveLogin(Map<String, dynamic> data) async {
    final sp = await _sp;
    await sp.setString('accessToken', '${data['accessToken'] ?? ''}');
    await sp.setString('refreshToken', '${data['refreshToken'] ?? ''}');
    await sp.setString('permissions', jsonEncode(data['permissions'] ?? []));
    await sp.setString('user', jsonEncode(data['user'] ?? {}));
    await sp.setBool('mustChangePassword', data['mustChangePassword'] == true);
  }

  static Future<void> clearMustChange() async {
    await (await _sp).setBool('mustChangePassword', false);
  }

  static Future<void> clear() async => (await _sp).clear();
}
