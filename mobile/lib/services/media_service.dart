import 'dart:io';

import 'package:http/http.dart' as http;
import 'package:image_picker/image_picker.dart';
import 'package:permission_handler/permission_handler.dart';

import 'api.dart';

class MediaService {
  static final _picker = ImagePicker();

  static Future<bool> ensureCamera() async {
    var s = await Permission.camera.status;
    if (s.isGranted) return true;
    s = await Permission.camera.request();
    return s.isGranted;
  }

  /// Take a photo; returns local file path or null if cancelled/denied.
  static Future<String?> takePhoto() async {
    final ok = await ensureCamera();
    if (!ok) throw StateError('需要相机权限才能拍照');
    final file = await _picker.pickImage(source: ImageSource.camera, imageQuality: 85, maxWidth: 1600);
    return file?.path;
  }

  /// Upload local file via MinIO presign; returns objectKey. Falls back to local path if MinIO unavailable.
  static Future<String> uploadPhoto(String localPath) async {
    try {
      final data = await Api.post('/media/upload-url', {
        'contentType': 'image/jpeg',
        'ext': 'jpg',
      });
      final uploadUrl = '${data['uploadUrl'] ?? ''}';
      final objectKey = '${data['objectKey'] ?? ''}';
      if (uploadUrl.isEmpty || objectKey.isEmpty) {
        return localPath;
      }
      final bytes = await File(localPath).readAsBytes();
      final res = await http.put(
        Uri.parse(uploadUrl),
        headers: {'Content-Type': 'image/jpeg'},
        body: bytes,
      );
      if (res.statusCode >= 300) {
        throw StateError('上传失败 HTTP ${res.statusCode}');
      }
      return objectKey;
    } on ApiException catch (e) {
      // MinIO down: keep local path so Outbox can still record attempt
      if (e.code.contains('UPSTREAM') || e.message.contains('minio')) {
        return localPath;
      }
      rethrow;
    }
  }
}
