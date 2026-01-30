import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/voice_service.dart';

class VoiceButton extends StatelessWidget {
  final Function(String) onResult;

  const VoiceButton({
    super.key,
    required this.onResult,
  });

  @override
  Widget build(BuildContext context) {
    return Consumer<VoiceService>(
      builder: (context, voice, _) {
        // Show download progress
        if (voice.isDownloading) {
          return Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerHighest,
              shape: BoxShape.circle,
            ),
            child: const Padding(
              padding: EdgeInsets.all(12),
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
          );
        }

        return GestureDetector(
          onLongPressStart: (_) => _startListening(context, voice),
          onLongPressEnd: (_) => _stopListening(voice),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            width: voice.isListening ? 64 : 48,
            height: voice.isListening ? 64 : 48,
            decoration: BoxDecoration(
              color: voice.isListening
                  ? Theme.of(context).colorScheme.error
                  : Theme.of(context).colorScheme.primary,
              shape: BoxShape.circle,
              boxShadow: voice.isListening
                  ? [
                      BoxShadow(
                        color: Theme.of(context).colorScheme.error.withOpacity(0.5),
                        blurRadius: 16,
                        spreadRadius: 4,
                      ),
                    ]
                  : null,
            ),
            child: Stack(
              alignment: Alignment.center,
              children: [
                // Pulsing animation when listening
                if (voice.isListening)
                  TweenAnimationBuilder<double>(
                    tween: Tween(begin: 1.0, end: 1.3),
                    duration: const Duration(milliseconds: 500),
                    curve: Curves.easeInOut,
                    builder: (context, scale, child) {
                      return Transform.scale(
                        scale: scale,
                        child: Container(
                          width: 64,
                          height: 64,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            border: Border.all(
                              color: Theme.of(context).colorScheme.error.withOpacity(0.3),
                              width: 2,
                            ),
                          ),
                        ),
                      );
                    },
                    onEnd: () {
                      // Restart animation
                    },
                  ),

                // Icon
                Icon(
                  voice.isListening ? Icons.mic : Icons.mic_none,
                  color: Colors.white,
                  size: voice.isListening ? 32 : 24,
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  void _startListening(BuildContext context, VoiceService voice) async {
    if (!voice.isAvailable) {
      final initialized = await voice.initialize();
      if (!initialized) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(voice.error ?? 'Voice input not available'),
            backgroundColor: Colors.red,
          ),
        );
        return;
      }
    }

    await voice.startListening(onResult: onResult);
  }

  void _stopListening(VoiceService voice) {
    voice.stopListening();
  }
}

/// Floating version of the voice button for alternative UI
class FloatingVoiceButton extends StatelessWidget {
  final Function(String) onResult;

  const FloatingVoiceButton({
    super.key,
    required this.onResult,
  });

  @override
  Widget build(BuildContext context) {
    return Consumer<VoiceService>(
      builder: (context, voice, _) {
        return FloatingActionButton.large(
          onPressed: () async {
            if (voice.isListening) {
              voice.stopListening();
            } else {
              if (!voice.isAvailable) {
                await voice.initialize();
              }
              await voice.startListening(onResult: onResult);
            }
          },
          backgroundColor: voice.isListening
              ? Theme.of(context).colorScheme.error
              : Theme.of(context).colorScheme.primary,
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                voice.isListening ? Icons.stop : Icons.mic,
                size: 32,
              ),
              if (voice.isListening && voice.currentText.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text(
                    '...',
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
