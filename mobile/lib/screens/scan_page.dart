import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:permission_handler/permission_handler.dart';

/// Full-screen scanner; pops with scanned string.
class ScanPage extends StatefulWidget {
  const ScanPage({super.key, this.title = '扫码'});
  final String title;

  @override
  State<ScanPage> createState() => _ScanPageState();
}

class _ScanPageState extends State<ScanPage> {
  final controller = MobileScannerController(detectionSpeed: DetectionSpeed.normal);
  bool _torch = false;
  bool _handled = false;
  String? _err;
  final manual = TextEditingController();

  @override
  void initState() {
    super.initState();
    _prep();
  }

  Future<void> _prep() async {
    final s = await Permission.camera.request();
    if (!s.isGranted) {
      setState(() => _err = '未授予相机权限，可手输唯一码');
    }
  }

  void _onDetect(BarcodeCapture cap) {
    if (_handled) return;
    final raw = cap.barcodes.isEmpty ? null : cap.barcodes.first.rawValue;
    if (raw == null || raw.isEmpty) return;
    _handled = true;
    HapticFeedback.mediumImpact();
    Navigator.of(context).pop(raw);
  }

  @override
  void dispose() {
    controller.dispose();
    manual.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        title: Text(widget.title),
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        actions: [
          IconButton(
            icon: Icon(_torch ? Icons.flash_on : Icons.flash_off),
            onPressed: () async {
              await controller.toggleTorch();
              setState(() => _torch = !_torch);
            },
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: _err != null
                ? Center(child: Text(_err!, style: const TextStyle(color: Colors.white70)))
                : Stack(
                    fit: StackFit.expand,
                    children: [
                      MobileScanner(controller: controller, onDetect: _onDetect),
                      IgnorePointer(
                        child: CustomPaint(painter: _FramePainter()),
                      ),
                    ],
                  ),
          ),
          Container(
            color: const Color(0xFF102A43),
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: manual,
                    style: const TextStyle(color: Colors.white),
                    decoration: const InputDecoration(
                      hintText: '扫码失败可手输唯一码',
                      hintStyle: TextStyle(color: Colors.white54),
                      border: OutlineInputBorder(),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: () {
                    final v = manual.text.trim();
                    if (v.isEmpty) return;
                    Navigator.of(context).pop(v);
                  },
                  child: const Text('确定'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _FramePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final rect = Rect.fromCenter(
      center: Offset(size.width / 2, size.height / 2),
      width: size.width * 0.7,
      height: size.width * 0.7,
    );
    final overlay = Path()..addRect(Offset.zero & size);
    final cut = Path()..addRRect(RRect.fromRectXY(rect, 12, 12));
    final path = Path.combine(PathOperation.difference, overlay, cut);
    canvas.drawPath(path, Paint()..color = Colors.black54);
    final border = Paint()
      ..color = const Color(0xFF2BB0ED)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3;
    canvas.drawRRect(RRect.fromRectXY(rect, 12, 12), border);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
