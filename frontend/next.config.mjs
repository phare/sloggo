import createMDX from "@next/mdx";

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "export",
  pageExtensions: ["ts", "tsx", "js", "jsx", "md", "mdx"],
  // REMINDER: new react compiler to memoize the components
  // https://react.dev/learn/react-compiler
  reactCompiler: true,
};

const withMDX = createMDX({
  extension: /\.mdx?$/,
  options: {
    remarkPlugins: [],
    // REMINDER: plugin names must be strings so they can be serialized and
    // used by the Turbopack MDX loader.
    rehypePlugins: [["rehype-pretty-code", { keepBackground: false }]],
  },
});

export default withMDX(nextConfig);
