-- Add EVPN fabric configuration parameters to expected_switch.
-- esi_base_mac: per-switch base MAC for computing ESI (Ethernet Segment
--   Identifier) in MLAG deployments. Nullable; absent means no ESI override.
-- df_preference: Designated Forwarder election preference (0–65535).
--   Higher value = preferred DF. Nullable; absent means use switch default.
ALTER TABLE public.expected_switches
    ADD COLUMN esi_base_mac macaddr,
    ADD COLUMN df_preference smallint;
