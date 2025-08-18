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

// Construct expected download URL
const fileName = `lash_${version}_${mappedPlatform}_${mappedArch}`;
const archiveExt = platform === 'win32' ? 'zip' : 'tar.gz';
const expectedUrl = `https://github.com/lacymorrow/lash/releases/download/v${version}/${fileName}.${archiveExt}`;

console.log(`Downloading lash v${version} for ${platform} ${arch}...`);

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
    const request = https.get(url, {
      headers: {
        'User-Agent': 'lash-cli-installer',
        'Accept': 'application/octet-stream'
      }
    }, (response) => {
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
    });
    request.on('error', reject);
  });
}

async function getAssetUrlFromGitHubReleaseTag(versionString) {
  const apiUrl = `https://api.github.com/repos/lacymorrow/lash/releases/tags/v${versionString}`;
  const responseBody = await new Promise((resolve, reject) => {
    const req = https.get(apiUrl, {
      headers: {
        'User-Agent': 'lash-cli-installer',
        'Accept': 'application/vnd.github+json'
      }
    }, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        if (res.statusCode !== 200) {
          reject(new Error(`GitHub API error ${res.statusCode}`));
          return;
        }
        resolve(data);
      });
    });
    req.on('error', reject);
  });

  let json;
  try {
    json = JSON.parse(responseBody);
  } catch (e) {
    throw new Error('Failed to parse GitHub release JSON');
  }

  if (!json.assets || !Array.isArray(json.assets)) {
    throw new Error('GitHub release contains no assets');
  }

  // Build matchers for OS and Arch across common aliases
  const osAliases = [mappedPlatform, mappedPlatform.toLowerCase()];
  if (mappedPlatform === 'Darwin') osAliases.push('macOS', 'macos', 'darwin');
  if (mappedPlatform === 'Windows') osAliases.push('win32', 'windows');
  if (mappedPlatform === 'Linux') osAliases.push('linux');

  const archAliases = [mappedArch];
  if (mappedArch === 'x86_64') archAliases.push('amd64');
  if (mappedArch === 'arm64') archAliases.push('aarch64');

  const candidates = json.assets.filter((asset) => {
    const name = asset.name || '';
    const hasOs = osAliases.some((alias) => name.toLowerCase().includes(String(alias).toLowerCase()));
    const hasArch = archAliases.some((alias) => name.toLowerCase().includes(String(alias).toLowerCase()));
    const hasExt = name.endsWith(`.${archiveExt}`);
    const hasVersion = name.includes(versionString);
    return hasOs && hasArch && hasExt && hasVersion;
  });

  if (candidates.length === 0) {
    throw new Error('No matching asset found for your platform/arch');
  }

  // Prefer exactly expected naming if present
  const exactName = `${fileName}.${archiveExt}`;
  const exact = candidates.find((a) => a.name === exactName);
  const chosen = exact || candidates[0];
  return chosen.browser_download_url;
}

function findFileRecursively(rootDir, targetFileName) {
  const stack = [rootDir];
  while (stack.length) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    for (const entry of entries) {
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(full);
      } else if (entry.isFile() && entry.name === targetFileName) {
        return full;
      }
    }
  }
  return null;
}

async function install() {
  try {
    let urlToUse = expectedUrl;
    try {
      console.log(`Trying URL: ${urlToUse}`);
      await download(urlToUse, tempFile);
    } catch (err) {
      if (String(err.message || '').includes('Failed to download: 404')) {
        console.log('Asset not found at expected URL, querying GitHub release assets...');
        try {
          urlToUse = await getAssetUrlFromGitHubReleaseTag(version);
          console.log(`Resolved asset URL: ${urlToUse}`);
          await download(urlToUse, tempFile);
        } catch (resolveErr) {
          console.error(`Could not resolve asset from GitHub: ${resolveErr.message}`);
          throw err; // rethrow original to trigger fallback
        }
      } else {
        throw err;
      }
    }
    
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
    const finalBinary = path.join(binDir, binaryName);
    const extractedBinary = findFileRecursively(extractDir, binaryName);
    if (extractedBinary && fs.existsSync(extractedBinary)) {
      fs.copyFileSync(extractedBinary, finalBinary);
      if (platform !== 'win32') {
        fs.chmodSync(finalBinary, 0o755);
      }
      console.log(`✅ lash v${version} installed successfully!`);
      console.log(`Binary location: ${finalBinary}`);
    } else {
      throw new Error('Binary not found after extraction');
    }
    
    // Cleanup
    fs.rmSync(tempFile, { force: true });
    fs.rmSync(extractDir, { recursive: true, force: true });
    
  } catch (error) {
    console.error('Installation failed:', error.message);
    // Fallback: try to install from source with Go toolchain
    try {
      console.log('Attempting fallback: building from source with Go...');
      const moduleRef = `github.com/lacymorrow/lash@v${version}`;
      execSync(`go install ${moduleRef}`, { stdio: 'inherit' });
      const goBin = execSync('go env GOPATH').toString().trim() + (platform === 'win32' ? '\\bin\\' : '/bin/');
      const srcBinary = path.join(goBin, platform === 'win32' ? 'lash.exe' : 'lash');
      if (fs.existsSync(srcBinary)) {
        const finalBinary = path.join(binDir, platform === 'win32' ? 'lash.exe' : 'lash');
        fs.copyFileSync(srcBinary, finalBinary);
        if (platform !== 'win32') {
          fs.chmodSync(finalBinary, 0o755);
        }
        console.log(`✅ lash v${version} installed from source successfully!`);
        console.log(`Binary location: ${finalBinary}`);
        process.exit(0);
      } else {
        throw new Error('Go-built binary not found in GOPATH/bin');
      }
    } catch (fallbackErr) {
      console.error('Fallback installation failed:', fallbackErr.message);
      process.exit(1);
    }
  }
}

install();