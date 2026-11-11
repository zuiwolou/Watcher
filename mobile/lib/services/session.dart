import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

class Session {
  static const _secure = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );

  static Future<SharedPreferences> get _sp => SharedPreferences.getInstance();

  static Future<String?> accessToken() async => _secure.read(key: 'accessToken');
  static Future<String?> refreshToken() async => _secure.read(key: 'refreshToken');

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
    await _secure.write(key: 'accessToken', value: '${data['accessToken'] ?? ''}');
    await _secure.write(key: 'refreshToken', value: '${data['refreshToken'] ?? ''}');
    await sp.setString('permissions', jsonEncode(data['permissions'] ?? []));
    await sp.setString('user', jsonEncode(data['user'] ?? {}));
    await sp.setBool('mustChangePassword', data['mustChangePassword'] == true);
    // migrate away from legacy plaintext prefs if present
    await sp.remove('accessToken');
    await sp.remove('refreshToken');
  }

  static Future<void> clearMustChange() async {
    await (await _sp).setBool('mustChangePassword', false);
  }

  static Future<void> clearCacheMeta() async {
    final sp = await _sp;
    await sp.remove('cacheTimeAlarms');
    await sp.remove('cacheTimeHome');
  }

  static Future<void> setCacheTime(String key) async {
    await (await _sp).setString(key, DateTime.now().toIso8601String());
  }

  static Future<String?> cacheTime(String key) async => (await _sp).getString(key);

  static Future<void> clear() async {
    await _secure.deleteAll();
    await (await _sp).clear();
  }
}
