import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/settings_service.dart';
import '../services/api_service.dart';

class LLMSettingsSection extends StatefulWidget {
  const LLMSettingsSection({super.key});

  @override
  State<LLMSettingsSection> createState() => _LLMSettingsSectionState();
}

class _LLMSettingsSectionState extends State<LLMSettingsSection> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadProviders());
  }

  Future<void> _loadProviders() async {
    final settings = context.read<SettingsService>();
    final api = context.read<ApiService>();

    if (api.isConnected && api.baseUrl != null) {
      await settings.fetchProviders(api.baseUrl!);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Consumer2<SettingsService, ApiService>(
      builder: (context, settings, api, _) {
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      'LLM Provider',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    if (settings.isLoading)
                      const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                  ],
                ),
                const SizedBox(height: 12),

                // Provider dropdown
                DropdownButtonFormField<String>(
                  value: settings.selectedProvider,
                  decoration: const InputDecoration(
                    labelText: 'Provider',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  items: settings.availableProviders.map((provider) {
                    return DropdownMenuItem(
                      value: provider.name,
                      child: Text(provider.displayName),
                    );
                  }).toList(),
                  onChanged: settings.isLoading
                      ? null
                      : (value) async {
                          if (value != null && api.baseUrl != null) {
                            await settings.selectProvider(value);
                            await settings.fetchModels(api.baseUrl!, value);
                          }
                        },
                ),
                const SizedBox(height: 16),

                // Model dropdown
                if (settings.selectedProvider != null) ...[
                  DropdownButtonFormField<String>(
                    value: settings.selectedModel,
                    decoration: const InputDecoration(
                      labelText: 'Model',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    items: settings.availableModels.map((model) {
                      return DropdownMenuItem(
                        value: model.id,
                        child: Text(
                          model.name,
                          overflow: TextOverflow.ellipsis,
                        ),
                      );
                    }).toList(),
                    onChanged: settings.isLoading
                        ? null
                        : (value) async {
                            if (value != null) {
                              await settings.selectModel(value);
                              api.setProviderOverride(
                                settings.selectedProvider,
                                settings.selectedModel,
                              );
                            }
                          },
                  ),
                  const SizedBox(height: 8),
                  OutlinedButton.icon(
                    onPressed: settings.isLoading || api.baseUrl == null
                        ? null
                        : () async {
                            await settings.fetchModels(
                              api.baseUrl!,
                              settings.selectedProvider!,
                            );
                          },
                    icon: const Icon(Icons.refresh, size: 16),
                    label: const Text('Refresh Models'),
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 8,
                      ),
                    ),
                  ),
                ],

                if (settings.error != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      settings.error!,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                        fontSize: 12,
                      ),
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
