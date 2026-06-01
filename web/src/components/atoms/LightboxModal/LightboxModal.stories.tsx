import type { Meta, StoryObj } from "@storybook/react";
import { LightboxModal } from "./LightboxModal";

const meta: Meta<typeof LightboxModal> = {
  title: "Atoms/LightboxModal",
  component: LightboxModal,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof LightboxModal>;

export const ImageLightbox: Story = {
  args: {
    open: true,
    onOpenChange: () => {},
    imageSrc: "https://placehold.co/1200x800/1a1a1a/faf8f4?text=Full+Resolution+Image",
    imageAlt: "Sample image",
  },
};

export const MermaidLightbox: Story = {
  args: {
    open: true,
    onOpenChange: () => {},
    svgHtml: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 200" width="400" height="200">
      <rect x="10" y="10" width="180" height="40" rx="5" fill="#f0ede8" stroke="#1a1a1a" stroke-width="1"/>
      <text x="100" y="35" text-anchor="middle" font-size="14" fill="#1a1a1a">Parser</text>
      <rect x="210" y="10" width="180" height="40" rx="5" fill="#f0ede8" stroke="#1a1a1a" stroke-width="1"/>
      <text x="300" y="35" text-anchor="middle" font-size="14" fill="#1a1a1a">Renderer</text>
      <line x1="190" y1="30" x2="210" y2="30" stroke="#1a1a1a" stroke-width="1" marker-end="url(#arrow)"/>
      <defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#1a1a1a"/></marker></defs>
      <rect x="100" y="100" width="200" height="40" rx="5" fill="#f0ede8" stroke="#1a1a1a" stroke-width="1"/>
      <text x="200" y="125" text-anchor="middle" font-size="14" fill="#1a1a1a">Output</text>
      <line x1="100" y1="50" x2="200" y2="100" stroke="#1a1a1a" stroke-width="1" marker-end="url(#arrow)"/>
      <line x1="300" y1="50" x2="200" y2="100" stroke="#1a1a1a" stroke-width="1" marker-end="url(#arrow)"/>
    </svg>`,
  },
};

export const Closed: Story = {
  args: {
    open: false,
    onOpenChange: () => {},
    imageSrc: "https://placehold.co/600x400",
  },
};
