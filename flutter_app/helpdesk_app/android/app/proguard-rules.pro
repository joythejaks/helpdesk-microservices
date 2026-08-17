# The Flutter Gradle plugin automatically includes the engine's own
# consumer proguard rules (keeps io.flutter.** intact) whenever
# minifyEnabled is true — no need to duplicate that here.

# Play Core split-install classes referenced by Flutter's deferred-
# components support, which this app doesn't use, but flutter.jar
# references them regardless — keeps R8 from warning/failing on missing
# classes for a feature that's simply never invoked.
-dontwarn com.google.android.play.core.**
-keep class com.google.android.play.core.** { *; }
