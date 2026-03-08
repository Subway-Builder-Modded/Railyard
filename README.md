<p align="center">
  <img src="build/appicon.png" width="128" height="128" alt="Railyard">
</p>

<h1 align="center">Railyard</h1>

<p align="center">
  A mod and map manager for <a href="https://store.steampowered.com/app/1078000/Subway_Builder/">Subway Builder</a>.
</p>

## Features

- **Map Browser**: Search and install community-made maps from the Railyard registry.
- **Mod Management**: Install, enable, and disable mods for Subway Builder.
- **Game Launcher**: Launch Subway Builder directly from Railyard with mods and maps loaded automatically.
- **Map Thumbnails**: Auto-generated SVG thumbnails rendered from PMTiles vector data.
- **Live Logs**: Stream and view game console output in real time.

To download and install Railyard, visit the [download page](https://subwaybuildermodded.com/railyard).

## Development prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js](https://nodejs.org/) (LTS recommended)
- [pnpm](https://pnpm.io/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

## Getting Started

```bash
# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Run in development mode (hot reload)
wails dev

# Build for production
wails build
```

## How It Works

1. **Registry** — Railyard clones a Git-based registry of available maps and mods.
2. **Installation** — Maps are downloaded as zip archives containing PMTiles, config, and GeoJSON data files. These are extracted to the Railyard data directory.
3. **Mod Generation** — At game launch, Railyard generates a Subway Builder mod (`com.railyard.maploader`) that registers all installed maps with the game's modding API.
4. **Tile Serving** — A local PMTiles server starts on a random port to serve vector tiles to the game at runtime.
5. **Thumbnails** — SVG thumbnails are generated from water layer features in the map's vector tiles and cached for display in the UI.

## License

See [LICENSE](LICENSE) for details.
