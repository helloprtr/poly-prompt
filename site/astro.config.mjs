import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";

export default defineConfig({
  site: "https://helloprtr.github.io/poly-prompt",
  base: "/poly-prompt",
  output: "static",
  integrations: [sitemap()],
});
