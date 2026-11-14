import L from 'leaflet';
import { MapContainer, Marker, Polyline, Popup, TileLayer } from 'react-leaflet';
import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';
import 'leaflet/dist/leaflet.css';

const defaultIcon = L.icon({
  iconUrl: markerIcon,
  iconRetinaUrl: markerIcon2x,
  shadowUrl: markerShadow,
  iconSize: [25, 41],
  iconAnchor: [12, 41],
  popupAnchor: [1, -34],
  shadowSize: [41, 41],
});
L.Marker.prototype.options.icon = defaultIcon;

export interface MapEndpoint {
  siteId: string;
  latitude: string;
  longitude: string;
  municipio?: string | null;
  estado?: string | null;
}

interface Props {
  pointA: MapEndpoint;
  pointB: MapEndpoint;
  distanceKm?: number | null;
}

function label(endpoint: MapEndpoint): string {
  const location = [endpoint.municipio, endpoint.estado].filter(Boolean).join('/');
  return `${endpoint.siteId}${location ? ` — ${location}` : ''}`;
}

function distanceBetween(positionA: [number, number], positionB: [number, number]): number {
  const toRadians = (value: number) => (value * Math.PI) / 180;
  const earthRadiusKm = 6371;
  const deltaLat = toRadians(positionB[0] - positionA[0]);
  const deltaLng = toRadians(positionB[1] - positionA[1]);
  const a =
    Math.sin(deltaLat / 2) ** 2 +
    Math.cos(toRadians(positionA[0])) * Math.cos(toRadians(positionB[0])) * Math.sin(deltaLng / 2) ** 2;
  return 2 * earthRadiusKm * Math.asin(Math.min(1, Math.sqrt(a)));
}

function distanceLabel(value: number): string {
  return `${value.toLocaleString('pt-BR', { maximumFractionDigits: 1 })} km`;
}

export function RadioLinkMap({ pointA, pointB, distanceKm }: Props) {
  const positionA: [number, number] = [Number(pointA.latitude), Number(pointA.longitude)];
  const positionB: [number, number] = [Number(pointB.latitude), Number(pointB.longitude)];
  const midpoint: [number, number] = [
    (positionA[0] + positionB[0]) / 2,
    (positionA[1] + positionB[1]) / 2,
  ];
  const distance = distanceKm != null && distanceKm > 0 ? distanceKm : distanceBetween(positionA, positionB);
  const pill = L.divIcon({
    className: 'radio-link-map__pill',
    html: `<span>${distanceLabel(distance)}</span>`,
    iconSize: [0, 0],
    iconAnchor: [0, 0],
  });

  return (
    <MapContainer
      bounds={[positionA, positionB]}
      boundsOptions={{ padding: [48, 48] }}
      scrollWheelZoom={false}
      className="radio-link-map__leaflet"
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      <Marker position={positionA}>
        <Popup>Ponta A — {label(pointA)}</Popup>
      </Marker>
      <Marker position={positionB}>
        <Popup>Ponta B — {label(pointB)}</Popup>
      </Marker>
      <Polyline
        positions={[positionA, positionB]}
        pathOptions={{ color: '#0f62fe', weight: 3, dashArray: '10 8', opacity: 0.9 }}
      />
      <Marker position={midpoint} icon={pill} interactive={false} />
    </MapContainer>
  );
}
