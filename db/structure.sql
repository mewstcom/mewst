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
-- Name: account_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    account_id uuid NOT NULL,
    profile_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    phone_number character varying NOT NULL,
    locale character varying NOT NULL,
    sign_in_count integer DEFAULT 0 NOT NULL,
    current_signed_in_at timestamp without time zone,
    last_signed_in_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


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
-- Name: organization_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_members (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    organization_id uuid NOT NULL,
    account_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: organization_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    organization_id uuid NOT NULL,
    profile_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: organizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizations (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: phone_number_verifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.phone_number_verifications (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    phone_number character varying NOT NULL,
    raw_phone_number character varying NOT NULL,
    confirmation_code character varying NOT NULL,
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
-- Name: profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profilable_type character varying NOT NULL,
    atname public.citext NOT NULL,
    name character varying DEFAULT ''::character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    avatar_data text,
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
-- Name: twitter_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.twitter_accounts (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    access_token character varying NOT NULL,
    scopes character varying[] NOT NULL,
    uid character varying NOT NULL,
    username character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: account_profiles account_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_profiles
    ADD CONSTRAINT account_profiles_pkey PRIMARY KEY (id);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


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
-- Name: organization_members organization_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_members
    ADD CONSTRAINT organization_members_pkey PRIMARY KEY (id);


--
-- Name: organization_profiles organization_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT organization_profiles_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: phone_number_verifications phone_number_verifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phone_number_verifications
    ADD CONSTRAINT phone_number_verifications_pkey PRIMARY KEY (id);


--
-- Name: posts posts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.posts
    ADD CONSTRAINT posts_pkey PRIMARY KEY (id);


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
-- Name: twitter_accounts twitter_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.twitter_accounts
    ADD CONSTRAINT twitter_accounts_pkey PRIMARY KEY (id);


--
-- Name: index_account_profiles_on_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_account_profiles_on_account_id ON public.account_profiles USING btree (account_id);


--
-- Name: index_account_profiles_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_account_profiles_on_profile_id ON public.account_profiles USING btree (profile_id);


--
-- Name: index_accounts_on_phone_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_accounts_on_phone_number ON public.accounts USING btree (phone_number);


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
-- Name: index_organization_members_on_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_organization_members_on_account_id ON public.organization_members USING btree (account_id);


--
-- Name: index_organization_members_on_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_organization_members_on_organization_id ON public.organization_members USING btree (organization_id);


--
-- Name: index_organization_members_on_organization_id_and_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_organization_members_on_organization_id_and_account_id ON public.organization_members USING btree (organization_id, account_id);


--
-- Name: index_organization_profiles_on_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_organization_profiles_on_organization_id ON public.organization_profiles USING btree (organization_id);


--
-- Name: index_organization_profiles_on_organization_id_and_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_organization_profiles_on_organization_id_and_profile_id ON public.organization_profiles USING btree (organization_id, profile_id);


--
-- Name: index_organization_profiles_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_organization_profiles_on_profile_id ON public.organization_profiles USING btree (profile_id);


--
-- Name: index_phone_number_verifications_on_pn_and_cc; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_phone_number_verifications_on_pn_and_cc ON public.phone_number_verifications USING btree (phone_number, confirmation_code);


--
-- Name: index_posts_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_posts_on_profile_id ON public.posts USING btree (profile_id);


--
-- Name: index_profiles_on_atname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_profiles_on_atname ON public.profiles USING btree (atname);


--
-- Name: index_twitter_accounts_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_twitter_accounts_on_profile_id ON public.twitter_accounts USING btree (profile_id);


--
-- Name: follows fk_rails_5e22b9865a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_5e22b9865a FOREIGN KEY (source_profile_id) REFERENCES public.profiles(id);


--
-- Name: account_profiles fk_rails_85b5ae6af1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_profiles
    ADD CONSTRAINT fk_rails_85b5ae6af1 FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: organization_profiles fk_rails_87611e9014; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT fk_rails_87611e9014 FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: organization_members fk_rails_9e0fce095d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_members
    ADD CONSTRAINT fk_rails_9e0fce095d FOREIGN KEY (account_id) REFERENCES public.accounts(id);


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
-- Name: organization_profiles fk_rails_d0c5dfda7d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT fk_rails_d0c5dfda7d FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: twitter_accounts fk_rails_eb2a6ad196; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.twitter_accounts
    ADD CONSTRAINT fk_rails_eb2a6ad196 FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: account_profiles fk_rails_f84ed5bded; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_profiles
    ADD CONSTRAINT fk_rails_f84ed5bded FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: organization_members fk_rails_ff629e24d8; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_members
    ADD CONSTRAINT fk_rails_ff629e24d8 FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20220000000001'),
('20221119004702');


