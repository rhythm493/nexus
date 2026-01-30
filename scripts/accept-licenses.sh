#!/bin/bash
# Accept Android SDK licenses

sudo mkdir -p /usr/lib/android-sdk/licenses

echo "24333f8a63b6825ea9c5514f83c2829b004d1fee" | sudo tee /usr/lib/android-sdk/licenses/android-sdk-license
echo "d975f751698a77b662f1254ddbeed3901e976f5a" | sudo tee /usr/lib/android-sdk/licenses/intel-android-extra-license
echo "84831b9409646a918e30573bab4c9c91346d8abd" | sudo tee /usr/lib/android-sdk/licenses/android-sdk-preview-license

echo "Done! Licenses accepted."
