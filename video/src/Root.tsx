import { Composition } from "remotion";
import { NexusDemo } from "./NexusDemo";

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="NexusDemo"
        component={NexusDemo}
        durationInFrames={750}
        fps={30}
        width={1080}
        height={1920}
        defaultProps={{ layout: "vertical" as const }}
      />
      <Composition
        id="NexusDemoHorizontal"
        component={NexusDemo}
        durationInFrames={750}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ layout: "horizontal" as const }}
      />
    </>
  );
};
