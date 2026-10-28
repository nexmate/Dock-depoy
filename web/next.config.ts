import type { NextConfig } from "next"

// Rewrites are resolved at build time; the Docker build sets API_URL=http://api:8080.
const apiUrl = process.env.API_URL ?? "http://localhost:8080"

const nextConfig: NextConfig = {
  output: "standalone",
  // Compression would buffer Server-Sent Events from the API.
  compress: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiUrl}/api/:path*` }]
  },
}

export default nextConfig
