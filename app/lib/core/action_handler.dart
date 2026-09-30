/// Handles actions dispatched by interactive components.
///
/// When a user taps a button, submits a form, or changes a toggle,
/// the component fires an action with a name and optional arguments.
/// Implementations forward this to the server via the action API.
abstract class ActionHandler {
  /// Execute [action] with optional [args].
  ///
  /// Returns true if the action was dispatched successfully.
  Future<bool> handleAction(String action, Map<String, dynamic>? args);
}
