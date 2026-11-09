import 'package:image_picker/image_picker.dart';
import 'package:permission_handler/permission_handler.dart';

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
}
