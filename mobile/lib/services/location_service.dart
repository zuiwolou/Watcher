import 'package:geolocator/geolocator.dart';
import 'package:permission_handler/permission_handler.dart';

class LocationFix {
  LocationFix({required this.lat, required this.lng, required this.accuracy});
  final double lat;
  final double lng;
  final double accuracy;
}

class LocationService {
  static Future<bool> ensurePermission() async {
    var status = await Permission.locationWhenInUse.status;
    if (status.isGranted) return true;
    status = await Permission.locationWhenInUse.request();
    if (!status.isGranted) return false;

    final enabled = await Geolocator.isLocationServiceEnabled();
    if (!enabled) {
      await Geolocator.openLocationSettings();
      return false;
    }
    return true;
  }

  /// Returns null if permission denied; throws if position cannot be obtained.
  static Future<LocationFix?> current({bool required = false}) async {
    final ok = await ensurePermission();
    if (!ok) {
      if (required) {
        throw StateError('需要定位权限才能提交，请在系统设置中开启');
      }
      return null;
    }
    final pos = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(accuracy: LocationAccuracy.high, timeLimit: Duration(seconds: 12)),
    );
    return LocationFix(lat: pos.latitude, lng: pos.longitude, accuracy: pos.accuracy);
  }
}
