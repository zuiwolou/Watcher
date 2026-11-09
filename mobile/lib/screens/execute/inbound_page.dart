import 'package:flutter/material.dart';

import 'exec_forms.dart';

class InboundPage extends StatefulWidget {
  const InboundPage({super.key});
  @override
  State<InboundPage> createState() => _InboundPageState();
}

class _InboundPageState extends State<InboundPage> with ExecSubmitMixin {
  final orderId = TextEditingController();
  final code = TextEditingController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('入库确认')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(controller: orderId, decoration: const InputDecoration(labelText: '入库单ID'), keyboardType: TextInputType.number),
          TextField(
            controller: code,
            decoration: InputDecoration(
              labelText: '唯一码',
              suffixIcon: IconButton(
                icon: const Icon(Icons.qr_code_scanner),
                onPressed: () async {
                  final v = await openScan(context, title: '入库扫码');
                  if (v != null) setState(() => code.text = v);
                },
              ),
            ),
          ),
          LocationPhotoBar(
            fix: fix,
            photoPath: photoPath,
            onLocate: () => refreshLocation(),
            onPhoto: capturePhoto,
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 48,
            child: FilledButton(
              onPressed: busy
                  ? null
                  : () => runSubmit(
                        path: '/inbound/orders/${orderId.text}/confirm',
                        body: {'uniqueCode': code.text},
                        label: '入库确认',
                      ),
              child: const Text('提交确认'),
            ),
          ),
          ExecResultBanner(msg: msg, isError: err),
        ],
      ),
    );
  }
}
