import 'package:flutter/material.dart';

import 'exec_forms.dart';

class OutboundPage extends StatefulWidget {
  const OutboundPage({super.key});
  @override
  State<OutboundPage> createState() => _OutboundPageState();
}

class _OutboundPageState extends State<OutboundPage> with ExecSubmitMixin {
  final orderId = TextEditingController();
  final code = TextEditingController();
  String action = 'lock';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('出库锁定 / 签收')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(controller: orderId, decoration: const InputDecoration(labelText: '出库单ID'), keyboardType: TextInputType.number),
          TextField(
            controller: code,
            decoration: InputDecoration(
              labelText: '唯一码',
              suffixIcon: IconButton(
                icon: const Icon(Icons.qr_code_scanner),
                onPressed: () async {
                  final v = await openScan(context, title: '出库扫码');
                  if (v != null) setState(() => code.text = v);
                },
              ),
            ),
          ),
          DropdownButtonFormField<String>(
            initialValue: action,
            items: const [
              DropdownMenuItem(value: 'lock', child: Text('锁定')),
              DropdownMenuItem(value: 'receive', child: Text('签收')),
            ],
            onChanged: (v) => setState(() => action = v ?? 'lock'),
            decoration: const InputDecoration(labelText: '动作'),
          ),
          LocationPhotoBar(
            fix: fix,
            photoPath: photoPath,
            onLocate: () => refreshLocation(required: action == 'receive'),
            onPhoto: capturePhoto,
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 48,
            child: FilledButton(
              onPressed: busy
                  ? null
                  : () => runSubmit(
                        path: '/outbound/orders/${orderId.text}/confirm',
                        body: {'uniqueCode': code.text, 'action': action},
                        label: action == 'lock' ? '出库锁定' : '签收',
                        requireLocation: action == 'receive',
                      ),
              child: Text(action == 'lock' ? '提交锁定' : '提交签收'),
            ),
          ),
          ExecResultBanner(msg: msg, isError: err),
        ],
      ),
    );
  }
}
