import 'dart:io';

import 'package:path_provider/path_provider.dart';

/// Writes [bytes] to a file named [filename] under the app's documents
/// directory. Returns the full saved path.
Future<String> saveBytes(String filename, List<int> bytes) async {
  final dir = await getApplicationDocumentsDirectory();
  final file = File('${dir.path}/$filename');
  await file.writeAsBytes(bytes);
  return file.path;
}
