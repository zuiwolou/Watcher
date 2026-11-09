import 'package:flutter/material.dart';

import 'exec_forms.dart';

class InspectionPage extends StatefulWidget {
  const InspectionPage({super.key});
  @override
  State<InspectionPage> createState() => _InspectionPageState();
}

class _InspectionPageState extends State<InspectionPage> with ExecSubmitMixin {
  final taskId = TextEditingController();
  final pointId = TextEditingController();
  String status = 'done';
  final remark = TextEditingController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('巡检打卡')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(controller: taskId, decoration: const InputDecoration(labelText: '巡检任务ID'), keyboardType: TextInputType.number),
          TextField(controller: pointId, decoration: const InputDecoration(labelText: '点位ID'), keyboardType: TextInputType.number),
          DropdownButtonFormField<String>(
            initialValue: status,
            items: const [
              DropdownMenuItem(value: 'done', child: Text('正常')),
              DropdownMenuItem(value: 'abnormal', child: Text('异常')),
            ],
            onChanged: (v) => setState(() => status = v ?? 'done'),
            decoration: const InputDecoration(labelText: '结果'),
          ),
          if (status == 'abnormal')
            TextField(controller: remark, decoration: const InputDecoration(labelText: '异常说明')),
          LocationPhotoBar(
            fix: fix,
            photoPath: photoPath,
            onLocate: () => refreshLocation(required: true),
            onPhoto: capturePhoto,
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 48,
            child: FilledButton(
              onPressed: busy
                  ? null
                  : () => runSubmit(
                        path: '/inspection/tasks/${taskId.text}/checkin',
                        body: {
                          'pointId': int.tryParse(pointId.text) ?? 0,
                          'status': status,
                          'remark': remark.text,
                        },
                        label: '巡检打卡',
                        requireLocation: true,
                      ),
              child: const Text('打卡'),
            ),
          ),
          ExecResultBanner(msg: msg, isError: err),
        ],
      ),
    );
  }
}
