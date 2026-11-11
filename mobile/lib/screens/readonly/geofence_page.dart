import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../services/api.dart';

/// Read-only geofence overview: list + OSM leaflet webview (Baidu later).
class GeofencePage extends StatefulWidget {
  const GeofencePage({super.key});
  @override
  State<GeofencePage> createState() => _GeofencePageState();
}

class _GeofencePageState extends State<GeofencePage> {
  List fences = [];
  String? err;
  late final WebViewController _web;

  @override
  void initState() {
    super.initState();
    _web = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..loadHtmlString(_mapHtml());
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await Api.get('/geofences');
      fences = data is List ? data : [];
      setState(() {});
    } catch (e) {
      setState(() => err = e.toString());
    }
  }

  String _mapHtml() => '''
<!DOCTYPE html><html><head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<style>html,body,#m{height:100%;margin:0}</style>
</head><body><div id="m"></div>
<script>
var map=L.map('m').setView([31.23,121.47],11);
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19}).addTo(map);
</script></body></html>
''';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('围栏/地图（只读）')),
      body: Column(
        children: [
          SizedBox(height: 220, child: WebViewWidget(controller: _web)),
          if (err != null) Padding(padding: const EdgeInsets.all(8), child: Text(err!, style: const TextStyle(color: Colors.red))),
          const Padding(
            padding: EdgeInsets.all(12),
            child: Align(alignment: Alignment.centerLeft, child: Text('围栏列表', style: TextStyle(fontWeight: FontWeight.w600))),
          ),
          Expanded(
            child: ListView(
              children: fences.map((e) {
                final m = e as Map;
                return ListTile(
                  title: Text('${m['Name'] ?? '-'}'),
                  subtitle: Text('${m['ShapeType']} · 仓 ${m['WarehouseID']} · ${m['Enabled'] == true ? '启用' : '停用'}'),
                );
              }).toList(),
            ),
          ),
        ],
      ),
    );
  }
}
