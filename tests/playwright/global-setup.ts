import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

async function globalSetup(config: FullConfig) {
  console.log('✅ Global setup is running');

  // Define source and destination paths
  // from /caddy-data/caddy/pki/authorities/local/root.crt
  // to /usr/local/share/ca-certificates/*
  const certSource = '/caddy-data/caddy/pki/authorities/local/root.crt';
  const certDest = '/usr/local/share/ca-certificates/caddy-root.crt';

  // Copy the certificate
  try {
    fs.copyFileSync(certSource, certDest);
    console.log(`📁 Copied cert from ${certSource} to ${certDest}`);
  } catch (err) {
    console.error(`❌ Failed to copy cert: ${err}`);
    throw err;
  }

  // Copy the certificate
  try {
    fs.copyFileSync(
        '/caddy-data/caddy/certificates/local/lettrapp.aliases.containernetwork/lettrapp.aliases.containernetwork.crt',
        '/usr/local/share/ca-certificates/lettrapp.aliases.containernetwork.crt',
    );
    console.log(`📁 Copied cert from TODO to TODO`);
  } catch (err) {
    console.error(`❌ Failed to copy cert: ${err}`);
    throw err;
  }

  

  // Update system trust store
  try {
    execSync('update-ca-certificates', { stdio: 'inherit' });
    console.log('🔐 Certificate added to system trust store');
  } catch (err) {
    console.error(`❌ Failed to update CA certificates: ${err}`);
    throw err;
  }
}

export default globalSetup;
