import 'package:flutter/material.dart';

import '../../services/location_service.dart';
import '../../services/media_service.dart';
import '../../services/outbox.dart';
import '../scan_page.dart';

Future<String?> openScan(BuildContext context, {String title = '扫码'}) {
  return Navigator.of(context).push<String>(MaterialPageRoute(builder: (_) => ScanPage(title: title)));
}

class ExecResultBanner extends StatelessWidget {
  const ExecResultBanner({super.key, required this.msg, this.isError = false});
  final String msg;
  final bool isError;
  @override
  Widget build(BuildContext context) {
    if (msg.isEmpty) return const SizedBox.shrink();
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(top: 12),
      padding: const EdgeInsets.all(12),
      color: isError ? Colors.red.shade50 : Colors.green.shade50,
      child: Text(msg, style: TextStyle(color: isError ? Colors.red.shade800 : Colors.green.shade800)),
    );
  }
}

mixin ExecSubmitMixin<T extends StatefulWidget> on State<T> {
  String msg = '';
  bool err = false;
  bool busy = false;
  String? photoPath;
  LocationFix? fix;

  Future<void> capturePhoto() async {
    try {
      photoPath = await MediaService.takePhoto();
      setState(() {});
    } catch (e) {
      setState(() {
        err = true;
        msg = e.toString();
      });
    }
  }

  Future<void> refreshLocation({bool required = false}) async {
    try {
      fix = await LocationService.current(required: required);
      setState(() {});
    } catch (e) {
      setState(() {
        err = true;
        msg = e.toString();
      });
      rethrow;
    }
  }

  Future<void> runSubmit({
    required String path,
    required Map<String, dynamic> body,
    required String label,
    bool requireLocation = false,
  }) async {
    setState(() {
      busy = true;
      msg = '';
      err = false;
    });
    try {
      if (requireLocation) {
        await refreshLocation(required: true);
      } else {
        fix ??= await LocationService.current();
      }
      if (fix != null) {
        body['lat'] = fix!.lat;
        body['lng'] = fix!.lng;
      }
      if (photoPath != null) {
        body['photoObjectKey'] = photoPath; // local path until MinIO upload wired
      }
      final res = await OutboxService.instance.submitOrEnqueue(path: path, body: body, label: label);
      if (res['queued'] == true) {
        setState(() {
          msg = '网络异常，已写入离线 Outbox，待同步';
          err = false;
        });
      } else {
        setState(() {
          msg = '$label 成功';
          err = false;
        });
      }
    } catch (e) {
      setState(() {
        err = true;
        msg = e.toString();
      });
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }
}

class LocationPhotoBar extends StatelessWidget {
  const LocationPhotoBar({
    super.key,
    required this.fix,
    required this.photoPath,
    required this.onLocate,
    required this.onPhoto,
  });

  final LocationFix? fix;
  final String? photoPath;
  final VoidCallback onLocate;
  final VoidCallback onPhoto;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                fix == null
                    ? '定位：未获取'
                    : '定位：${fix!.lat.toStringAsFixed(5)}, ${fix!.lng.toStringAsFixed(5)} (±${fix!.accuracy.toStringAsFixed(0)}m)',
                style: const TextStyle(fontSize: 13),
              ),
            ),
            TextButton(onPressed: onLocate, child: const Text('定位')),
          ],
        ),
        Row(
          children: [
            Expanded(child: Text(photoPath == null ? '照片：未拍摄' : '照片：已拍', style: const TextStyle(fontSize: 13))),
            TextButton(onPressed: onPhoto, child: const Text('拍照')),
          ],
        ),
      ],
    );
  }
}
