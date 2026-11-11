import 'package:flutter/material.dart';

import '../../services/location_service.dart';
import '../../services/media_service.dart';
import '../../services/outbox.dart';
import '../scan_page.dart';

Future<String?> openScan(BuildContext context, {String title = '扫码'}) {
  return Navigator.of(context).push<String>(MaterialPageRoute(builder: (_) => ScanPage(title: title)));
}

Future<void> openContinuousScan(
  BuildContext context, {
  required String title,
  required Future<void> Function(String code) onCode,
  String? progressText,
}) {
  return Navigator.of(context).push(
    MaterialPageRoute(
      builder: (_) => ScanPage(
        title: title,
        continuous: true,
        onCode: onCode,
        progressText: progressText,
      ),
    ),
  );
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
  String? photoObjectKey;
  LocationFix? fix;

  /// Max acceptable GPS accuracy in meters for forced check-ins.
  static const double maxAccuracyM = 50;

  Future<void> capturePhoto() async {
    try {
      photoPath = await MediaService.takePhoto();
      photoObjectKey = null;
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
      if (required && fix != null && fix!.accuracy > maxAccuracyM) {
        throw StateError('定位精度不足（${fix!.accuracy.toStringAsFixed(0)}m > ${maxAccuracyM.toStringAsFixed(0)}m），请到开阔处重试');
      }
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
    bool requirePhoto = false,
  }) async {
    setState(() {
      busy = true;
      msg = '';
      err = false;
    });
    try {
      if (requirePhoto && photoPath == null) {
        throw StateError('本节点必须拍照后才能提交');
      }
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
        photoObjectKey ??= await MediaService.uploadPhoto(photoPath!);
        body['photoObjectKey'] = photoObjectKey;
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
      rethrow;
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
    this.requirePhotoHint = false,
  });

  final LocationFix? fix;
  final String? photoPath;
  final VoidCallback onLocate;
  final VoidCallback onPhoto;
  final bool requirePhotoHint;

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
            Expanded(
              child: Text(
                photoPath == null
                    ? (requirePhotoHint ? '照片：必拍（未拍摄）' : '照片：未拍摄')
                    : '照片：已拍',
                style: TextStyle(fontSize: 13, color: requirePhotoHint && photoPath == null ? Colors.orange.shade800 : null),
              ),
            ),
            TextButton(onPressed: onPhoto, child: const Text('拍照')),
          ],
        ),
      ],
    );
  }
}

/// Shared pending-order / task list picker.
class TaskPickerList extends StatelessWidget {
  const TaskPickerList({
    super.key,
    required this.items,
    required this.titleBuilder,
    required this.subtitleBuilder,
    required this.onSelect,
    this.emptyText = '暂无待办',
  });

  final List items;
  final String Function(Map row) titleBuilder;
  final String Function(Map row) subtitleBuilder;
  final void Function(Map row) onSelect;
  final String emptyText;

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) {
      return Padding(
        padding: const EdgeInsets.all(24),
        child: Text(emptyText, style: const TextStyle(color: Colors.black54)),
      );
    }
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: items.length,
      separatorBuilder: (_, __) => const Divider(height: 1),
      itemBuilder: (_, i) {
        final row = Map<String, dynamic>.from(items[i] as Map);
        return ListTile(
          title: Text(titleBuilder(row)),
          subtitle: Text(subtitleBuilder(row)),
          trailing: const Icon(Icons.chevron_right),
          onTap: () => onSelect(row),
        );
      },
    );
  }
}
