import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/location_service.dart';

class LocationSettingsSection extends StatelessWidget {
  const LocationSettingsSection({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<LocationService>(
      builder: (context, locationService, _) {
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Section header
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Row(
                children: [
                  const Icon(Icons.location_on, size: 20),
                  const SizedBox(width: 8),
                  Text(
                    'Location (for Grocery Mode)',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ],
              ),
            ),

            // Current location display
            ListTile(
              leading: Icon(
                locationService.hasLocation ? Icons.check_circle : Icons.error,
                color: locationService.hasLocation ? Colors.green : Colors.orange,
              ),
              title: const Text('Current Location'),
              subtitle: Text(
                locationService.hasLocation
                    ? locationService.getLocationSummary()
                    : 'No location set',
              ),
              trailing: locationService.isLoading
                  ? const SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : null,
            ),

            // Error message
            if (locationService.error != null)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Text(
                  locationService.error!,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                    fontSize: 12,
                  ),
                ),
              ),

            // Action buttons
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  // Get current location button
                  ElevatedButton.icon(
                    onPressed: locationService.isLoading
                        ? null
                        : () async {
                            await locationService.getCurrentLocation(
                              forceRefresh: true,
                            );
                          },
                    icon: const Icon(Icons.my_location, size: 18),
                    label: const Text('Get Current Location'),
                  ),

                  // Manual location button
                  OutlinedButton.icon(
                    onPressed: () => _showManualLocationDialog(context),
                    icon: const Icon(Icons.edit_location, size: 18),
                    label: const Text('Set Manually'),
                  ),

                  // Clear location button
                  if (locationService.hasLocation)
                    TextButton.icon(
                      onPressed: () async {
                        await locationService.clearLocation();
                      },
                      icon: const Icon(Icons.clear, size: 18),
                      label: const Text('Clear'),
                    ),
                ],
              ),
            ),

            // Permission info
            if (!locationService.permissionGranted)
              Padding(
                padding: const EdgeInsets.all(16),
                child: Card(
                  color: Colors.orange.shade900.withValues(alpha: 0.3),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Row(
                      children: [
                        const Icon(Icons.info, size: 20, color: Colors.orange),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              const Text(
                                'Location Permission Required',
                                style: TextStyle(fontWeight: FontWeight.bold),
                              ),
                              const SizedBox(height: 4),
                              const Text(
                                'Grant location access to use grocery price comparison features.',
                                style: TextStyle(fontSize: 12),
                              ),
                              const SizedBox(height: 8),
                              TextButton(
                                onPressed: () async {
                                  await locationService.requestPermission();
                                },
                                child: const Text('Grant Permission'),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),

            const Divider(),
          ],
        );
      },
    );
  }

  void _showManualLocationDialog(BuildContext context) {
    final latController = TextEditingController();
    final lonController = TextEditingController();
    final nameController = TextEditingController();

    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Set Location Manually'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: latController,
              decoration: const InputDecoration(
                labelText: 'Latitude',
                hintText: 'e.g., 12.9716',
              ),
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: lonController,
              decoration: const InputDecoration(
                labelText: 'Longitude',
                hintText: 'e.g., 77.5946',
              ),
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: nameController,
              decoration: const InputDecoration(
                labelText: 'Location Name (optional)',
                hintText: 'e.g., Bangalore',
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () async {
              final lat = double.tryParse(latController.text);
              final lon = double.tryParse(lonController.text);

              if (lat != null && lon != null) {
                final locationService = context.read<LocationService>();
                await locationService.setLocation(
                  lat,
                  lon,
                  nameController.text.isEmpty
                      ? '${lat.toStringAsFixed(4)}, ${lon.toStringAsFixed(4)}'
                      : nameController.text,
                );
                if (context.mounted) {
                  Navigator.pop(context);
                }
              }
            },
            child: const Text('Set'),
          ),
        ],
      ),
    );
  }
}
