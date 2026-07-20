// This file is only ever compiled for the web target, selected via the
// conditional export in file_saver.dart — dart:html here is intentional,
// not an accidental non-web import.
// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:html' as html;

/// Triggers a normal browser download of [bytes] named [filename]. Returns
/// [filename] once the download has been kicked off (the browser owns the
/// rest of the flow — there's no further completion signal to await).
Future<String> saveBytes(String filename, List<int> bytes) async {
  final blob = html.Blob([bytes]);
  final url = html.Url.createObjectUrlFromBlob(blob);
  html.AnchorElement(href: url)
    ..setAttribute('download', filename)
    ..click();
  html.Url.revokeObjectUrl(url);
  return filename;
}
