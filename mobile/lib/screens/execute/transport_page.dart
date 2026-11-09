import 'package:flutter/material.dart';

import 'exec_forms.dart';

class TransportPage extends StatefulWidget {
  const TransportPage({super.key});
  @override
  State<TransportPage> createState() => _TransportPageState();
}

class _TransportPageState extends State<TransportPage> with ExecSubmitMixin {
  final taskId = TextEditingController();
  String node = 'waypoint';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('运输打卡')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(controller: taskId, decoration: const InputDecoration(labelText: '任务ID'), keyboardType: TextInputType.number),
          DropdownButtonFormField<String>(
            initialValue: node,
            items: const [
              DropdownMenuItem(value: 'depart', child: Text('发车')),
              DropdownMenuItem(value: 'waypoint', child: Text('途经')),
              DropdownMenuItem(value: 'arrive', child: Text('到达')),
            ],
            onChanged: (v) => setState(() => node = v ?? 'waypoint'),
            decoration: const InputDecoration(labelText: '节点'),
          ),
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
                        path: '/transport/checkins',
                        body: {
                          'taskId': int.tryParse(taskId.text) ?? 0,
                          'nodeType': node,
                        },
                        label: '运输打卡',
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
