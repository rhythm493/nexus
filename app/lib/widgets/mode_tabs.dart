import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/mode_service.dart';
import '../services/api_service.dart';
import '../services/location_service.dart';
import 'loading_shimmer.dart';

class ModeTabs extends StatelessWidget {
  const ModeTabs({super.key});

  IconData _getIconData(String iconName) {
    // Map icon name strings to Material IconData
    switch (iconName) {
      case 'shopping_cart':
        return Icons.shopping_cart;
      case 'speaker':
        return Icons.speaker;
      case 'search':
        return Icons.search;
      case 'home':
        return Icons.home;
      case 'music_note':
        return Icons.music_note;
      case 'lightbulb':
        return Icons.lightbulb;
      default:
        return Icons.help_outline;
    }
  }

  Future<void> _handleModeChange(BuildContext context, String modeId) async {
    final modeService = context.read<ModeService>();
    final apiService = context.read<ApiService>();
    final locationService = context.read<LocationService>();

    // Show loading indicator
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Row(
            children: [
              const SizedBox(
                width: 16,
                height: 16,
                child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
              ),
              const SizedBox(width: 12),
              Text('Switching to ${_getModeName(context, modeId)}...'),
            ],
          ),
          duration: const Duration(seconds: 1),
        ),
      );
    }

    // Select the mode
    await modeService.selectMode(modeId);
    apiService.setMode(modeId);

    // If grocery mode, ensure location is set
    if (modeId == 'grocery') {
      if (!locationService.hasLocation) {
        // Try to get location
        final position = await locationService.getCurrentLocation();
        if (position == null) {
          if (!context.mounted) return;
          // Show snackbar to inform user
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: const Text('Please enable location for grocery mode'),
              action: SnackBarAction(
                label: 'Enable',
                onPressed: () async {
                  await locationService.requestPermission();
                  await locationService.getCurrentLocation(forceRefresh: true);
                },
              ),
              duration: const Duration(seconds: 5),
            ),
          );
        }
      } else {
        // We have location, send it to backend in background
        if (!context.mounted) return;
        _sendLocationToBackend(context);
      }
    }

    // Clear conversation when switching modes
    apiService.newConversation();
  }

  String _getModeName(BuildContext context, String modeId) {
    final modeService = context.read<ModeService>();
    final mode = modeService.availableModes.firstWhere(
      (m) => m.id == modeId,
      orElse: () => AppMode(id: modeId, name: modeId, icon: 'help', description: ''),
    );
    return mode.name;
  }

  Future<void> _sendLocationToBackend(BuildContext context) async {
    final apiService = context.read<ApiService>();
    final locationService = context.read<LocationService>();

    if (!locationService.hasLocation || apiService.baseUrl == null) {
      return;
    }

    final position = locationService.currentPosition!;
    final locationName = locationService.locationName ?? 'Current Location';

    // Send a background message to set location
    // The message will trigger the location_set tool call
    try {
      await for (final event in apiService.chat(
        'Set my delivery location to $locationName (${position.latitude}, ${position.longitude})',
      )) {
        // Just consume the stream silently
        if (event.type == 'done') break;
      }
    } catch (e) {
      // Silently fail - user can manually set location later
      debugPrint('Failed to set location on backend: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Consumer2<ModeService, ApiService>(
      builder: (context, modeService, apiService, _) {
        // Show shimmer while loading modes
        if (modeService.isLoading && modeService.availableModes.isEmpty) {
          return const ModeTabsShimmer();
        }

        // Show placeholder if modes not loaded yet
        if (modeService.availableModes.isEmpty && !apiService.isConnected) {
          return Container(
            height: 56,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
              border: Border(
                bottom: BorderSide(
                  color: Theme.of(context).dividerColor,
                  width: 1,
                ),
              ),
            ),
            child: Center(
              child: Text(
                'Connect to server to enable modes',
                style: TextStyle(
                  color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.5),
                  fontSize: 12,
                ),
              ),
            ),
          );
        }

        if (modeService.availableModes.isEmpty) {
          return const SizedBox.shrink();
        }

        return Container(
          height: 56,
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context).dividerColor,
                width: 1,
              ),
            ),
          ),
          child: ListView.builder(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 8),
            itemCount: modeService.availableModes.length,
            itemBuilder: (context, index) {
              final mode = modeService.availableModes[index];
              final isSelected = mode.id == modeService.selectedModeId;

              return Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 8),
                child: ChoiceChip(
                  label: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        _getIconData(mode.icon),
                        size: 18,
                        color: isSelected
                            ? Theme.of(context).colorScheme.onPrimary
                            : Theme.of(context).colorScheme.onSurface,
                      ),
                      const SizedBox(width: 8),
                      Text(mode.name),
                    ],
                  ),
                  selected: isSelected,
                  onSelected: (selected) async {
                    if (selected) {
                      await _handleModeChange(context, mode.id);
                    }
                  },
                  selectedColor: Theme.of(context).colorScheme.primary,
                  backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
                  labelStyle: TextStyle(
                    color: isSelected
                        ? Theme.of(context).colorScheme.onPrimary
                        : Theme.of(context).colorScheme.onSurface,
                  ),
                ),
              );
            },
          ),
        );
      },
    );
  }
}
