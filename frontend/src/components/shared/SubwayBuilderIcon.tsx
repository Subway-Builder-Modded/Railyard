interface SubwayBuilderIconProps {
  className?: string;
}

export function SubwayBuilderIcon({ className }: SubwayBuilderIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 256 256"
      shapeRendering="geometricPrecision"
      className={className}
    >
      <g fill="none" strokeLinecap="butt" strokeLinejoin="round">
        <path
          d="M22.576599 251.721517 V168.104451 A145.527852 145.527852 0 0 1 168.104451 22.576599 H251.717312"
          stroke="#000000"
          strokeWidth="49.574072"
        />
        <path
          d="M22.576599 251.721517 V168.104451 A145.527852 145.527852 0 0 1 168.104451 22.576599 H251.717312"
          stroke="#d82233"
          strokeWidth="39.581582"
        />
        <path
          d="M67.800973 251.721517 V168.104451 A100.303478 100.303478 0 0 1 168.104451 67.800973 H251.717312"
          stroke="#000000"
          strokeWidth="49.574072"
        />
        <path
          d="M67.800973 251.721517 V168.104451 A100.303478 100.303478 0 0 1 168.104451 67.800973 H251.717312"
          stroke="#009952"
          strokeWidth="39.581582"
        />
        <path
          d="M111.362327 251.721517 V168.104451 A56.742124 56.742124 0 0 1 168.104451 111.362327 H251.717312"
          stroke="#000000"
          strokeWidth="49.574072"
        />
        <path
          d="M111.362327 251.721517 V168.104451 A56.742124 56.742124 0 0 1 168.104451 111.362327 H251.717312"
          stroke="#0062cf"
          strokeWidth="39.581582"
        />
      </g>
      <circle cx="171.100095" cy="22.576599" r="15.447398" fill="#ffffff" />
      <circle cx="171.100095" cy="67.800973" r="15.447398" fill="#ffffff" />
      <circle cx="171.100095" cy="111.362327" r="15.447398" fill="#ffffff" />
    </svg>
  );
}
