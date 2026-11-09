import 'package:flutter/material.dart';

import 'exec_forms.dart';

class InventoryPage extends StatefulWidget {
  const InventoryPage({super.key});
  @override
  State<InventoryPage> createState() => _InventoryPageState();
}

class _InventoryPageState extends State<InventoryPage> with ExecSubmitMixin {
  final taskId = TextEditingController();
  final code = TextEditingController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('盘点扫码')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(controller: taskId, decoration: const InputDecoration(labelText: '盘点任务ID'), keyboardType: TextInputType.number),
          TextField(
            controller: code,
            decoration: InputDecoration(
              labelText: '唯一码',
              suffixIcon: IconButton(
                icon: const Icon(Icons.qr_code_scanner),
                onPressed: () async {
                  final v = await openScan(context, title: '盘点扫码');
                  if (v != null) setState(() => code.text = v);
                },
              ),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 48,
            child: FilledButton(
              onPressed: busy
                  ? null
                  : () async {
                      await runSubmit(
                        path: '/inventory/tasks/${taskId.text}/scan',
                        body: {'uniqueCode': code.text},
                        label: '盘点扫码',
                      );
                      if (!err) code.clear();
                    },
              child: const Text('提交并继续'),
            ),
          ),
          ExecResultBanner(msg: msg, isError: err),
        ],
      ),
    );
  }
}
