import 'dart:convert';

import 'package:http/http.dart' as http;

import 'session.dart';

const apiBase = String.fromEnvironment('API_BASE', defaultValue: 'http://10.0.2.2:8080/api/v1');

class ApiException implements Exception {
  ApiException(this.code, this.message);
  final String code;
  final String message;
  @override
  String toString() => message.isEmpty ? code : message;
}

class Api {
  static Future<Map<String, String>> _headers({bool auth = true}) async {
    final headers = {'Content-Type': 'application/json', 'X-Client': 'app'};
    if (auth) {
      final t = await Session.accessToken();
      if (t != null && t.isNotEmpty) headers['Authorization'] = 'Bearer $t';
    }
    return headers;
  }

  static Future<dynamic> _handle(http.Response res) async {
    final body = jsonDecode(res.body) as Map<String, dynamic>;
    final code = '${body['code'] ?? ''}';
    if (code != 'OK') {
      throw ApiException(code, '${body['message'] ?? code}');
    }
    return body['data'];
  }

  static Future<Map<String, dynamic>> post(String path, Map body, {bool auth = true}) async {
    final res = await http.post(
      Uri.parse('$apiBase$path'),
      headers: await _headers(auth: auth),
      body: jsonEncode(body),
    );
    if (res.statusCode == 401 && auth) {
      final ok = await _tryRefresh();
      if (ok) {
        final retry = await http.post(
          Uri.parse('$apiBase$path'),
          headers: await _headers(auth: true),
          body: jsonEncode(body),
        );
        final data = await _handle(retry);
        return Map<String, dynamic>.from(data as Map? ?? {});
      }
    }
    final data = await _handle(res);
    return Map<String, dynamic>.from(data as Map? ?? {});
  }

  static Future<dynamic> get(String path) async {
    final res = await http.get(Uri.parse('$apiBase$path'), headers: await _headers());
    if (res.statusCode == 401) {
      final ok = await _tryRefresh();
      if (ok) {
        final retry = await http.get(Uri.parse('$apiBase$path'), headers: await _headers());
        return _handle(retry);
      }
    }
    return _handle(res);
  }

  static Future<bool> _tryRefresh() async {
    final rt = await Session.refreshToken();
    if (rt == null || rt.isEmpty) return false;
    try {
      final res = await http.post(
        Uri.parse('$apiBase/auth/refresh'),
        headers: {'Content-Type': 'application/json', 'X-Client': 'app'},
        body: jsonEncode({'refreshToken': rt}),
      );
      final body = jsonDecode(res.body) as Map<String, dynamic>;
      if (body['code'] != 'OK') return false;
      final data = body['data'] as Map<String, dynamic>;
      final perms = await Session.permissions();
      final user = await Session.user();
      await Session.saveLogin({
        'accessToken': data['accessToken'],
        'refreshToken': data['refreshToken'],
        'permissions': perms,
        'user': user,
        'mustChangePassword': false,
      });
      return true;
    } catch (_) {
      return false;
    }
  }
}
