--
-- PostgreSQL database dump
--

-- Dumped from database version 16.9
-- Dumped by pg_dump version 16.9

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: kimbap
--

-- *not* creating schema, since initdb creates it


ALTER SCHEMA public OWNER TO kimbap;

--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: kimbap
--

COMMENT ON SCHEMA public IS '';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: log; Type: TABLE; Schema: public; Owner: kimbap
--

CREATE TABLE public.log (
    id integer NOT NULL,
    "timestamp" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    user_id character varying(64) NOT NULL,
    type integer NOT NULL,
    request_content text,
    response_content text,
    error_content text,
    "serverID" character varying(128)
);


ALTER TABLE public.log OWNER TO kimbap;

--
-- Name: log_id_seq; Type: SEQUENCE; Schema: public; Owner: kimbap
--

CREATE SEQUENCE public.log_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.log_id_seq OWNER TO kimbap;

--
-- Name: log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kimbap
--

ALTER SEQUENCE public.log_id_seq OWNED BY public.log.id;


--
-- Name: mcp_events; Type: TABLE; Schema: public; Owner: kimbap
--

CREATE TABLE public.mcp_events (
    id integer NOT NULL,
    event_id character varying(255) NOT NULL,
    stream_id character varying(255) NOT NULL,
    session_id character varying(255) NOT NULL,
    message_type character varying(50) NOT NULL,
    message_data text NOT NULL,
    created_at timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp(3) without time zone NOT NULL
);


ALTER TABLE public.mcp_events OWNER TO kimbap;

--
-- Name: mcp_events_id_seq; Type: SEQUENCE; Schema: public; Owner: kimbap
--

CREATE SEQUENCE public.mcp_events_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.mcp_events_id_seq OWNER TO kimbap;

--
-- Name: mcp_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kimbap
--

ALTER SEQUENCE public.mcp_events_id_seq OWNED BY public.mcp_events.id;


--
-- Name: proxy; Type: TABLE; Schema: public; Owner: kimbap
--

CREATE TABLE public.proxy (
    id integer NOT NULL,
    name character varying(128) NOT NULL,
    addtime integer NOT NULL
);


ALTER TABLE public.proxy OWNER TO kimbap;

--
-- Name: proxy_id_seq; Type: SEQUENCE; Schema: public; Owner: kimbap
--

CREATE SEQUENCE public.proxy_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.proxy_id_seq OWNER TO kimbap;

--
-- Name: proxy_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kimbap
--

ALTER SEQUENCE public.proxy_id_seq OWNED BY public.proxy.id;


--
-- Name: server; Type: TABLE; Schema: public; Owner: kimbap
--

CREATE TABLE public.server (
    server_id character varying(128) NOT NULL,
    server_name character varying(128) NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    launch_config text NOT NULL,
    capabilities text NOT NULL,
    created_at timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp(3) without time zone NOT NULL,
    allow_user_input boolean DEFAULT false NOT NULL,
    proxy_id integer DEFAULT 0 NOT NULL,
    tool_tmpl_id character varying(128)
);


ALTER TABLE public.server OWNER TO kimbap;

--
-- Name: user; Type: TABLE; Schema: public; Owner: kimbap
--

CREATE TABLE public."user" (
    user_id character varying(64) NOT NULL,
    status integer NOT NULL,
    role integer NOT NULL,
    permissions text NOT NULL,
    server_api_keys text NOT NULL,
    ratelimit integer NOT NULL,
    name character varying(128) NOT NULL,
    encrypted_token text,
    proxy_id integer DEFAULT 0 NOT NULL,
    notes text,
    expires_at integer DEFAULT 0 NOT NULL,
    created_at integer DEFAULT 0 NOT NULL,
    updated_at integer DEFAULT 0 NOT NULL
);


ALTER TABLE public."user" OWNER TO kimbap;

--
-- Name: log id; Type: DEFAULT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.log ALTER COLUMN id SET DEFAULT nextval('public.log_id_seq'::regclass);


--
-- Name: mcp_events id; Type: DEFAULT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.mcp_events ALTER COLUMN id SET DEFAULT nextval('public.mcp_events_id_seq'::regclass);


--
-- Name: proxy id; Type: DEFAULT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.proxy ALTER COLUMN id SET DEFAULT nextval('public.proxy_id_seq'::regclass);


--
-- Name: log log_pkey; Type: CONSTRAINT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.log
    ADD CONSTRAINT log_pkey PRIMARY KEY (id);


--
-- Name: mcp_events mcp_events_pkey; Type: CONSTRAINT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.mcp_events
    ADD CONSTRAINT mcp_events_pkey PRIMARY KEY (id);


--
-- Name: proxy proxy_pkey; Type: CONSTRAINT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.proxy
    ADD CONSTRAINT proxy_pkey PRIMARY KEY (id);


--
-- Name: server server_pkey; Type: CONSTRAINT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public.server
    ADD CONSTRAINT server_pkey PRIMARY KEY (server_id);


--
-- Name: user user_pkey; Type: CONSTRAINT; Schema: public; Owner: kimbap
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (user_id);


--
-- Name: mcp_events_created_at_idx; Type: INDEX; Schema: public; Owner: kimbap
--

CREATE INDEX mcp_events_created_at_idx ON public.mcp_events USING btree (created_at);


--
-- Name: mcp_events_event_id_key; Type: INDEX; Schema: public; Owner: kimbap
--

CREATE UNIQUE INDEX mcp_events_event_id_key ON public.mcp_events USING btree (event_id);


--
-- Name: mcp_events_expires_at_idx; Type: INDEX; Schema: public; Owner: kimbap
--

CREATE INDEX mcp_events_expires_at_idx ON public.mcp_events USING btree (expires_at);


--
-- Name: mcp_events_session_id_idx; Type: INDEX; Schema: public; Owner: kimbap
--

CREATE INDEX mcp_events_session_id_idx ON public.mcp_events USING btree (session_id);


--
-- Name: mcp_events_stream_id_idx; Type: INDEX; Schema: public; Owner: kimbap
--

CREATE INDEX mcp_events_stream_id_idx ON public.mcp_events USING btree (stream_id);


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: kimbap
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;


--
-- PostgreSQL database dump complete
--

