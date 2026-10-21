-- BGP peer configuration for fabric switches.
-- Each row represents one BGP neighbor a switch should establish with
-- (typically spine switches for leaf-spine underlay, or peer leaves for MLAG).
-- The switch-config provider reads this table via NicoSwitchService to
-- generate the underlay BGP section of switch configs pushed via NVUE.

CREATE TABLE public.switch_bgp_peers (
    switch_bgp_peer_id uuid DEFAULT gen_random_uuid() NOT NULL,
    expected_switch_id uuid NOT NULL,
    peer_ip             inet NOT NULL,
    peer_asn            integer NOT NULL,
    vrf                 character varying(128) DEFAULT 'default' NOT NULL,
    peer_group          character varying(128),
    ttl                 integer DEFAULT 0 NOT NULL,
    source_interface    character varying(64),
    route_role          character varying(64),
    evpn_enabled        boolean DEFAULT false NOT NULL,
    -- bfd: Bidirectional Forwarding Detection, common in leaf-spine underlay BGP
    bfd_enabled         boolean DEFAULT false NOT NULL,
    created_at          timestamp with time zone DEFAULT now() NOT NULL,
    updated_at          timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.switch_bgp_peers
    ADD CONSTRAINT switch_bgp_peers_pkey PRIMARY KEY (switch_bgp_peer_id);

ALTER TABLE ONLY public.switch_bgp_peers
    ADD CONSTRAINT switch_bgp_peers_expected_switch_id_fkey
        FOREIGN KEY (expected_switch_id)
        REFERENCES public.expected_switches(expected_switch_id)
        ON DELETE CASCADE;

CREATE INDEX switch_bgp_peers_expected_switch_id_idx
    ON public.switch_bgp_peers USING btree (expected_switch_id);
