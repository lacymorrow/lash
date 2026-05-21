import type { ShipConfig } from "@lacymorrow/shipx"

export default {
	packageJsonPaths: [
		"packages/opencode/package.json",
		"package.json",
		"packages/app/package.json",
		"packages/core/package.json",
		"packages/desktop/package.json",
		"packages/enterprise/package.json",
		"packages/function/package.json",
		"packages/http-recorder/package.json",
		"packages/llm/package.json",
		"packages/plugin/package.json",
		"packages/script/package.json",
		"packages/sdk/js/package.json",
		"packages/slack/package.json",
		"packages/storybook/package.json",
		"packages/ui/package.json",
		"packages/web/package.json",
		"sdks/vscode/package.json",
	],
	bumpFiles: [
		{
			path: "packages/extensions/zed/extension.toml",
			pattern: /^version = "[^"]+"/m,
			replacement: (v: string) => `version = "${v}"`,
		},
		{
			path: "packages/extensions/zed/extension.toml",
			pattern: /releases\/download\/v[^/]+\//g,
			replacement: (v: string) => `releases/download/v${v}/`,
		},
	],
	git: {
		releaseBranch: "lash",
		commitMessage: "release: {tag}",
		commitFlags: "",
		pushFlags: "--force-with-lease",
	},
	npm: {
		cwd: "packages/opencode",
		access: "public",
	},
	steps: {
		test: false,
		cleanup: false,
		// ShipX doesn't support multi-package npm publish yet — lash needs
		// wrapper (lashcode) + N binary packages (lashcode-{platform}-{arch}).
		// Disable until ShipX supports multiple publish targets.
		npm: false,
		// ShipX creates GitHub releases from source tarballs. Lash needs to
		// upload pre-built binary archives. Disable until ShipX supports
		// custom release assets.
		githubRelease: false,
		// ShipX Homebrew step uses source tarball SHAs. Lash needs pre-built
		// binary URLs with per-platform SHA256. Disable until supported.
		homebrew: false,
	},
} satisfies ShipConfig
