#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const packageJson = require('../package.json');
const version = packageJson.version;

// Determine platform and architecture
const platform = process.platform;
const arch = process.arch;

// Map Node.js platform/arch to GoReleaser naming
const platformMap = {
  'darwin': 'Darwin',
  'linux': 'Linux',
  'win32': 'Windows'
};

const archMap = {
  'x64': 'x86_64',
  'arm64': 'arm64'
};

const mappedPlatform = platformMap[platform];
const mappedArch = archMap[arch];

if (!mappedPlatform || !mappedArch) {
  console.error(`Unsupported platform: ${platform} ${arch}`);
  process.exit(1);
}

// Construct download URL
const fileName = `lash_${version}_${mappedPlatform}_${mappedArch}`;
const archiveExt = platform === 'win32' ? 'zip' : 'tar.gz';
const downloadUrl = `https://github.com/lacymorrow/lash/releases/download/v${version}/${fileName}.${archiveExt}`;

console.log(`Downloading lash v${version} for ${platform} ${arch}...`);
console.log(`URL: ${downloadUrl}`);

// Create bin directory
const binDir = path.join(__dirname, '..', 'bin');
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Download and extract
const tempFile = path.join(binDir, `lash.${archiveExt}`);

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        return download(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode}`));
        return;
      }
      
      response.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', reject);
  });
}

async function install() {
  try {
    await download(downloadUrl, tempFile);
    
    // Extract the archive
    const extractDir = path.join(binDir, 'temp');
    if (!fs.existsSync(extractDir)) {
      fs.mkdirSync(extractDir, { recursive: true });
    }
    
    if (platform === 'win32') {
      // Extract zip (requires unzip or 7z)
      try {
        execSync(`powershell -command "Expand-Archive -Path '${tempFile}' -DestinationPath '${extractDir}' -Force"`, { stdio: 'inherit' });
      } catch (e) {
        console.error('Failed to extract with PowerShell, trying 7z...');
        execSync(`7z x "${tempFile}" -o"${extractDir}"`, { stdio: 'inherit' });
      }
    } else {
      // Extract tar.gz
      execSync(`tar -xzf "${tempFile}" -C "${extractDir}"`, { stdio: 'inherit' });
    }
    
    // Find and move the binary
    const binaryName = platform === 'win32' ? 'lash.exe' : 'lash';
    const extractedBinary = path.join(extractDir, fileName, binaryName);
    const finalBinary = path.join(binDir, binaryName);
    
    if (fs.existsSync(extractedBinary)) {
      fs.copyFileSync(extractedBinary, finalBinary);
      
      // Make executable on Unix systems
      if (platform !== 'win32') {
        fs.chmodSync(finalBinary, 0o755);
      }
      
      console.log(`✅ lash v${version} installed successfully!`);
      console.log(`Binary location: ${finalBinary}`);
    } else {
      throw new Error(`Binary not found in extracted archive: ${extractedBinary}`);
    }
    
    // Cleanup
    fs.rmSync(tempFile, { force: true });
    fs.rmSync(extractDir, { recursive: true, force: true });
    
  } catch (error) {
    console.error('Installation failed:', error.message);
    process.exit(1);
  }
}

install();