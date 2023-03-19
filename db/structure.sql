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
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: generate_ulid(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.generate_ulid() RETURNS uuid
    LANGUAGE sql
    AS $$
  SELECT (lpad(to_hex(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint), 12, '0') || encode(gen_random_bytes(10), 'hex'))::uuid;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: ar_internal_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ar_internal_metadata (
    key character varying NOT NULL,
    value character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: follows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.follows (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    source_profile_id uuid NOT NULL,
    target_profile_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: posts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.posts (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    content text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: profile_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.profile_members (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    atname public.citext NOT NULL,
    name character varying DEFAULT ''::character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    avatar_url character varying,
    deleted_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    password_digest character varying NOT NULL,
    locale character varying NOT NULL,
    sign_in_count integer DEFAULT 0 NOT NULL,
    current_signed_in_at timestamp without time zone,
    last_signed_in_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: verifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.verifications (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    event character varying NOT NULL,
    code character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: follows follows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT follows_pkey PRIMARY KEY (id);


--
-- Name: posts posts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.posts
    ADD CONSTRAINT posts_pkey PRIMARY KEY (id);


--
-- Name: profile_members profile_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profile_members
    ADD CONSTRAINT profile_members_pkey PRIMARY KEY (id);


--
-- Name: profiles profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: verifications verifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.verifications
    ADD CONSTRAINT verifications_pkey PRIMARY KEY (id);


--
-- Name: index_follows_on_source_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_follows_on_source_profile_id ON public.follows USING btree (source_profile_id);


--
-- Name: index_follows_on_source_profile_id_and_target_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_follows_on_source_profile_id_and_target_profile_id ON public.follows USING btree (source_profile_id, target_profile_id);


--
-- Name: index_follows_on_target_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_follows_on_target_profile_id ON public.follows USING btree (target_profile_id);


--
-- Name: index_posts_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_posts_on_profile_id ON public.posts USING btree (profile_id);


--
-- Name: index_profile_members_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_profile_members_on_profile_id ON public.profile_members USING btree (profile_id);


--
-- Name: index_profile_members_on_profile_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_profile_members_on_profile_id_and_user_id ON public.profile_members USING btree (profile_id, user_id);


--
-- Name: index_profile_members_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_profile_members_on_user_id ON public.profile_members USING btree (user_id);


--
-- Name: index_profiles_on_atname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_profiles_on_atname ON public.profiles USING btree (atname);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: index_verifications_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_verifications_on_created_at ON public.verifications USING btree (created_at);


--
-- Name: index_verifications_on_email_and_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_verifications_on_email_and_code ON public.verifications USING btree (email, code);


--
-- Name: profile_members fk_rails_5a8b59bd8b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profile_members
    ADD CONSTRAINT fk_rails_5a8b59bd8b FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: follows fk_rails_5e22b9865a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_5e22b9865a FOREIGN KEY (source_profile_id) REFERENCES public.profiles(id);


--
-- Name: profile_members fk_rails_87765715d2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profile_members
    ADD CONSTRAINT fk_rails_87765715d2 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: follows fk_rails_abf061b756; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_abf061b756 FOREIGN KEY (target_profile_id) REFERENCES public.profiles(id);


--
-- Name: posts fk_rails_cd61a4aa45; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.posts
    ADD CONSTRAINT fk_rails_cd61a4aa45 FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20220000000001'),
('20221119004702');


