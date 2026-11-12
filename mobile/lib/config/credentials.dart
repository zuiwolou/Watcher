/// 外部凭证。`--dart-define` 名称与开放平台申请的密钥名对齐。
///
/// ```bash
/// flutter run \
///   --dart-define=API_BASE=http://10.0.2.2:8080/api/v1 \
///   --dart-define=BAIDU_ANDROID_SDK_AK=你的Android_SDK_AK \
///   --dart-define=BAIDU_IOS_SDK_AK=你的iOS_SDK_AK \
///   --dart-define=MAP_PROVIDER=baidu
/// ```
class Credentials {
  Credentials._();

  static const apiBase = String.fromEnvironment(
    'API_BASE',
    defaultValue: 'http://10.0.2.2:8080/api/v1',
  );

  /// osm | baidu
  static const mapProvider = String.fromEnvironment(
    'MAP_PROVIDER',
    defaultValue: 'osm',
  );

  /// 百度地图开放平台 → Android SDK → AK
  static const baiduAndroidSdkAk = String.fromEnvironment(
    'BAIDU_ANDROID_SDK_AK',
    defaultValue: '',
  );

  /// 百度地图开放平台 → iOS SDK → AK
  static const baiduIosSdkAk = String.fromEnvironment(
    'BAIDU_IOS_SDK_AK',
    defaultValue: '',
  );

  /// 兼容旧 define；优先用分端 SDK AK
  static const _legacyBaiduMapAk = String.fromEnvironment(
    'BAIDU_MAP_AK',
    defaultValue: '',
  );

  static String get baiduSdkAk {
    if (baiduAndroidSdkAk.trim().isNotEmpty) return baiduAndroidSdkAk.trim();
    if (baiduIosSdkAk.trim().isNotEmpty) return baiduIosSdkAk.trim();
    return _legacyBaiduMapAk.trim();
  }

  static bool get useBaiduMap => mapProvider == 'baidu' && baiduSdkAk.isNotEmpty;
}
