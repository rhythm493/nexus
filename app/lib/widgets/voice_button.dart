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
          return _DownloadingIndicator(progress: voice.downloadProgress);
        }

        return GestureDetector(
          // Regular tap for download dialog, long press for voice
          onTap: voice.needsDownload ? () => _startListening(context, voice) : null,
          onLongPressStart: voice.needsDownload ? null : (_) => _startListening(context, voice),
          onLongPressEnd: voice.needsDownload ? null : (_) => _stopListening(voice),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            width: voice.isListening ? 64 : 48,
            height: voice.isListening ? 64 : 48,
            decoration: BoxDecoration(
              color: voice.isListening
                  ? Theme.of(context).colorScheme.error
                  : voice.needsDownload
                      ? Theme.of(context).colorScheme.surfaceContainerHighest
                      : Theme.of(context).colorScheme.primary,
              shape: BoxShape.circle,
              boxShadow: voice.isListening
                  ? [
                      BoxShadow(
                        color: Theme.of(context).colorScheme.error.withValues(alpha: 0.5),
                        blurRadius: 16,
                        spreadRadius: 4,
                      ),
                    ]
                  : null,
            ),
            child: Stack(
              alignment: Alignment.center,
              children: [
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
                              color: Theme.of(context).colorScheme.error.withValues(alpha: 0.3),
                              width: 2,
                            ),
                          ),
                        ),
                      );
                    },
                  ),
                Icon(
                  voice.needsDownload
                      ? Icons.download
                      : voice.isListening
                          ? Icons.mic
                          : Icons.mic_none,
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
    // If model needs download, show confirmation dialog
    if (voice.needsDownload) {
      final confirmed = await _showDownloadDialog(context);
      if (!confirmed) return;
      if (!context.mounted) return;

      final downloaded = await voice.downloadModel();
      if (!context.mounted) return;
      if (!downloaded) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(voice.error ?? 'Download failed'),
            backgroundColor: Colors.red,
          ),
        );
        return;
      }
    }

    if (!voice.isAvailable) {
      final initialized = await voice.initialize();
      if (!initialized) {
        if (!context.mounted) return;
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

  Future<bool> _showDownloadDialog(BuildContext context) async {
    return await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: const Text('Download Speech Model'),
            content: const Text(
              'Voice input requires a ~70MB speech recognition model.\n\n'
              'This will be downloaded once from your Nexus server and cached for future use.',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx, false),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: () => Navigator.pop(ctx, true),
                child: const Text('Download'),
              ),
            ],
          ),
        ) ??
        false;
  }

  void _stopListening(VoiceService voice) {
    voice.stopListening();
  }
}

class _DownloadingIndicator extends StatelessWidget {
  final double progress;

  const _DownloadingIndicator({required this.progress});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 48,
      height: 48,
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        shape: BoxShape.circle,
      ),
      child: Stack(
        alignment: Alignment.center,
        children: [
          SizedBox(
            width: 36,
            height: 36,
            child: CircularProgressIndicator(
              value: progress > 0 ? progress : null,
              strokeWidth: 2,
            ),
          ),
          Text(
            progress > 0 ? '${(progress * 100).toInt()}%' : '',
            style: const TextStyle(fontSize: 9),
          ),
        ],
      ),
    );
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
                const Padding(
                  padding: EdgeInsets.only(top: 4),
                  child: Text(
                    '...',
                    style: TextStyle(fontSize: 12),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
