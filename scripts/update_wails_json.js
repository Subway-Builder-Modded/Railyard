import * as fs from 'fs';

const config = JSON.parse(fs.readFileSync(import.meta.dirname + '/../wails.json', 'utf-8'));

const VERSION = process.env.VERSION;

if (!VERSION) {
    console.error("RAILYARD_VERSION environment variable is not set");
    process.exit(1);
}

config.info.productVersion = VERSION;

fs.writeFileSync(import.meta.dirname + '/../wails.json', JSON.stringify(config, null, 2));
console.log(`Updated wails.json with version ${VERSION}`);