/// Performance configuration constants for the app
library;

/// Chat screen performance
class ChatPerformance {
  /// Number of tokens to accumulate before updating UI during streaming
  static const int streamingUpdateThrottle = 5;

  /// Cache extent for ListView (pixels to cache off-screen)
  static const double listViewCacheExtent = 500.0;

  /// Scroll animation duration
  static const Duration scrollAnimationDuration = Duration(milliseconds: 200);
}

/// Network performance
class NetworkPerformance {
  /// Default timeout for API requests
  static const Duration defaultTimeout = Duration(seconds: 30);

  /// Health check timeout
  static const Duration healthCheckTimeout = Duration(seconds: 5);

  /// Mode fetch cache duration
  static const Duration modeCacheDuration = Duration(minutes: 5);

  /// Discovery auto-stop duration
  static const Duration discoveryStopDuration = Duration(seconds: 10);
}

/// UI performance
class UIPerformance {
  /// Shimmer animation duration
  static const Duration shimmerDuration = Duration(milliseconds: 1500);

  /// Typing indicator animation duration
  static const Duration typingIndicatorDuration = Duration(milliseconds: 1500);

  /// Snackbar display duration
  static const Duration snackbarDuration = Duration(seconds: 2);

  /// Loading snackbar duration
  static const Duration loadingSnackbarDuration = Duration(seconds: 1);
}
