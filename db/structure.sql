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
-- Name: comment_posts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comment_posts (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    post_id uuid NOT NULL,
    comment text NOT NULL,
    reposts_count integer DEFAULT 0 NOT NULL,
    stamps_count integer DEFAULT 0 NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: email_confirmations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_confirmations (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    event character varying NOT NULL,
    code character varying NOT NULL,
    succeeded_at timestamp without time zone,
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
    followed_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: oauth_access_grants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_access_grants (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    resource_owner_id uuid NOT NULL,
    user_id uuid NOT NULL,
    application_id uuid NOT NULL,
    token character varying NOT NULL,
    expires_in integer NOT NULL,
    redirect_uri text NOT NULL,
    scopes character varying DEFAULT ''::character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    revoked_at timestamp(6) without time zone
);


--
-- Name: oauth_access_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_access_tokens (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    resource_owner_id uuid,
    user_id uuid NOT NULL,
    application_id uuid NOT NULL,
    token character varying NOT NULL,
    refresh_token character varying,
    expires_in integer,
    scopes character varying,
    created_at timestamp(6) without time zone NOT NULL,
    revoked_at timestamp(6) without time zone,
    previous_refresh_token character varying DEFAULT ''::character varying NOT NULL
);


--
-- Name: oauth_applications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_applications (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    name character varying NOT NULL,
    uid character varying NOT NULL,
    secret character varying NOT NULL,
    redirect_uri text NOT NULL,
    scopes character varying DEFAULT ''::character varying NOT NULL,
    confidential boolean DEFAULT true NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: posts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.posts (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    kind character varying NOT NULL,
    published_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profileable_type character varying NOT NULL,
    profileable_id uuid NOT NULL,
    atname public.citext NOT NULL,
    name character varying DEFAULT ''::character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    avatar_url character varying DEFAULT ''::character varying NOT NULL,
    deleted_at timestamp without time zone,
    joined_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: reposts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reposts (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    post_id uuid NOT NULL,
    profile_id uuid NOT NULL,
    follow_id uuid NOT NULL,
    comment_post_id uuid NOT NULL,
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
-- Name: stamps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stamps (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    comment_post_id uuid NOT NULL,
    stamped_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
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
    signed_up_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: comment_posts comment_posts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_posts
    ADD CONSTRAINT comment_posts_pkey PRIMARY KEY (id);


--
-- Name: email_confirmations email_confirmations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_confirmations
    ADD CONSTRAINT email_confirmations_pkey PRIMARY KEY (id);


--
-- Name: follows follows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT follows_pkey PRIMARY KEY (id);


--
-- Name: oauth_access_grants oauth_access_grants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_grants
    ADD CONSTRAINT oauth_access_grants_pkey PRIMARY KEY (id);


--
-- Name: oauth_access_tokens oauth_access_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_tokens
    ADD CONSTRAINT oauth_access_tokens_pkey PRIMARY KEY (id);


--
-- Name: oauth_applications oauth_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_applications
    ADD CONSTRAINT oauth_applications_pkey PRIMARY KEY (id);


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
-- Name: reposts reposts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reposts
    ADD CONSTRAINT reposts_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: stamps stamps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamps
    ADD CONSTRAINT stamps_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: index_comment_posts_on_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_comment_posts_on_post_id ON public.comment_posts USING btree (post_id);


--
-- Name: index_email_confirmations_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_email_confirmations_on_created_at ON public.email_confirmations USING btree (created_at);


--
-- Name: index_email_confirmations_on_email_and_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_email_confirmations_on_email_and_code ON public.email_confirmations USING btree (email, code);


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
-- Name: index_oauth_access_grants_on_application_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_access_grants_on_application_id ON public.oauth_access_grants USING btree (application_id);


--
-- Name: index_oauth_access_grants_on_resource_owner_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_access_grants_on_resource_owner_id ON public.oauth_access_grants USING btree (resource_owner_id);


--
-- Name: index_oauth_access_grants_on_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_access_grants_on_token ON public.oauth_access_grants USING btree (token);


--
-- Name: index_oauth_access_grants_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_access_grants_on_user_id ON public.oauth_access_grants USING btree (user_id);


--
-- Name: index_oauth_access_tokens_on_application_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_access_tokens_on_application_id ON public.oauth_access_tokens USING btree (application_id);


--
-- Name: index_oauth_access_tokens_on_refresh_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_access_tokens_on_refresh_token ON public.oauth_access_tokens USING btree (refresh_token);


--
-- Name: index_oauth_access_tokens_on_resource_owner_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_access_tokens_on_resource_owner_id ON public.oauth_access_tokens USING btree (resource_owner_id);


--
-- Name: index_oauth_access_tokens_on_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_access_tokens_on_token ON public.oauth_access_tokens USING btree (token);


--
-- Name: index_oauth_access_tokens_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_access_tokens_on_user_id ON public.oauth_access_tokens USING btree (user_id);


--
-- Name: index_oauth_applications_on_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_applications_on_name ON public.oauth_applications USING btree (name);


--
-- Name: index_oauth_applications_on_uid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_applications_on_uid ON public.oauth_applications USING btree (uid);


--
-- Name: index_posts_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_posts_on_profile_id ON public.posts USING btree (profile_id);


--
-- Name: index_profiles_on_atname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_profiles_on_atname ON public.profiles USING btree (atname);


--
-- Name: index_profiles_on_profileable_type_and_profileable_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_profiles_on_profileable_type_and_profileable_id ON public.profiles USING btree (profileable_type, profileable_id);


--
-- Name: index_reposts_on_comment_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_reposts_on_comment_post_id ON public.reposts USING btree (comment_post_id);


--
-- Name: index_reposts_on_follow_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_reposts_on_follow_id ON public.reposts USING btree (follow_id);


--
-- Name: index_reposts_on_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_reposts_on_post_id ON public.reposts USING btree (post_id);


--
-- Name: index_reposts_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_reposts_on_profile_id ON public.reposts USING btree (profile_id);


--
-- Name: index_stamps_on_comment_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamps_on_comment_post_id ON public.stamps USING btree (comment_post_id);


--
-- Name: index_stamps_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamps_on_profile_id ON public.stamps USING btree (profile_id);


--
-- Name: index_stamps_on_profile_id_and_comment_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_stamps_on_profile_id_and_comment_post_id ON public.stamps USING btree (profile_id, comment_post_id);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: reposts fk_rails_0063d9ed2a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reposts
    ADD CONSTRAINT fk_rails_0063d9ed2a FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: stamps fk_rails_27da15755d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamps
    ADD CONSTRAINT fk_rails_27da15755d FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: oauth_access_grants fk_rails_330c32d8d9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_grants
    ADD CONSTRAINT fk_rails_330c32d8d9 FOREIGN KEY (resource_owner_id) REFERENCES public.profiles(id);


--
-- Name: comment_posts fk_rails_3c00337d07; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_posts
    ADD CONSTRAINT fk_rails_3c00337d07 FOREIGN KEY (post_id) REFERENCES public.posts(id);


--
-- Name: reposts fk_rails_5dbeb23937; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reposts
    ADD CONSTRAINT fk_rails_5dbeb23937 FOREIGN KEY (comment_post_id) REFERENCES public.comment_posts(id);


--
-- Name: follows fk_rails_5e22b9865a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_5e22b9865a FOREIGN KEY (source_profile_id) REFERENCES public.profiles(id);


--
-- Name: oauth_access_tokens fk_rails_732cb83ab7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_tokens
    ADD CONSTRAINT fk_rails_732cb83ab7 FOREIGN KEY (application_id) REFERENCES public.oauth_applications(id);


--
-- Name: oauth_access_tokens fk_rails_76012a03dc; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_tokens
    ADD CONSTRAINT fk_rails_76012a03dc FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: stamps fk_rails_9dee51665a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamps
    ADD CONSTRAINT fk_rails_9dee51665a FOREIGN KEY (comment_post_id) REFERENCES public.comment_posts(id);


--
-- Name: reposts fk_rails_9f623b101a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reposts
    ADD CONSTRAINT fk_rails_9f623b101a FOREIGN KEY (follow_id) REFERENCES public.follows(id);


--
-- Name: follows fk_rails_abf061b756; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_abf061b756 FOREIGN KEY (target_profile_id) REFERENCES public.profiles(id);


--
-- Name: oauth_access_grants fk_rails_b4b53e07b8; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_grants
    ADD CONSTRAINT fk_rails_b4b53e07b8 FOREIGN KEY (application_id) REFERENCES public.oauth_applications(id);


--
-- Name: posts fk_rails_cd61a4aa45; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.posts
    ADD CONSTRAINT fk_rails_cd61a4aa45 FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: reposts fk_rails_e9f75e0f73; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reposts
    ADD CONSTRAINT fk_rails_e9f75e0f73 FOREIGN KEY (post_id) REFERENCES public.posts(id);


--
-- Name: oauth_access_tokens fk_rails_ee63f25419; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_tokens
    ADD CONSTRAINT fk_rails_ee63f25419 FOREIGN KEY (resource_owner_id) REFERENCES public.profiles(id);


--
-- Name: oauth_access_grants fk_rails_f4d63eb352; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_grants
    ADD CONSTRAINT fk_rails_f4d63eb352 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20220000000001'),
('20221119004702');


