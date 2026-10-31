struct NodeFramework: Sendable {
    let id: String
    let packages: [String]
    let port: Int?

    static func detect(in dependencies: Set<String>) -> NodeFramework? {
        all.first { framework in framework.packages.contains { dependencies.contains($0) } }
    }

    static let all: [NodeFramework] = [
        NodeFramework(id: "nextjs", packages: ["next"], port: 3000),
        NodeFramework(id: "nuxt", packages: ["nuxt", "nuxt3"], port: 3000),
        NodeFramework(
            id: "remix", packages: ["@remix-run/react", "@remix-run/node", "@remix-run/serve"], port: 3000),
        NodeFramework(id: "sveltekit", packages: ["@sveltejs/kit"], port: 5173),
        NodeFramework(id: "solid-start", packages: ["solid-start", "@solidjs/start"], port: 3000),
        NodeFramework(id: "astro", packages: ["astro"], port: 4321),
        NodeFramework(id: "gatsby", packages: ["gatsby"], port: 8000),
        NodeFramework(id: "expo", packages: ["expo"], port: 8081),
        NodeFramework(id: "angular", packages: ["@angular/core"], port: 4200),
        NodeFramework(id: "svelte", packages: ["svelte"], port: 5173),
        NodeFramework(id: "solid", packages: ["solid-js"], port: 3000),
        NodeFramework(id: "react-native", packages: ["react-native"], port: 8081),
        NodeFramework(id: "cra", packages: ["react-scripts"], port: 3000),
        NodeFramework(
            id: "vite", packages: ["vite", "@vitejs/plugin-react", "@vitejs/plugin-vue"], port: 5173),
        NodeFramework(id: "nestjs", packages: ["@nestjs/core"], port: 3000),
        NodeFramework(id: "fastify", packages: ["fastify", "@fastify/core"], port: 3000),
        NodeFramework(id: "express", packages: ["express"], port: 3000),
        NodeFramework(id: "koa", packages: ["koa"], port: 3000),
        NodeFramework(id: "hapi", packages: ["@hapi/hapi"], port: 3000),
        NodeFramework(id: "strapi", packages: ["@strapi/strapi", "strapi"], port: 1337),
        NodeFramework(id: "payload", packages: ["payload"], port: 3000),
        NodeFramework(id: "electron", packages: ["electron"], port: nil),
    ]
}
