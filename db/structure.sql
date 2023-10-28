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
-- Name: actors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.actors (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    profile_id uuid NOT NULL,
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
-- Name: follow_notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.follow_notifications (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    notification_id uuid NOT NULL,
    source_profile_id uuid NOT NULL,
    target_profile_id uuid NOT NULL,
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
-- Name: good_job_batches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.good_job_batches (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    description text,
    serialized_properties jsonb,
    on_finish text,
    on_success text,
    on_discard text,
    callback_queue_name text,
    callback_priority integer,
    enqueued_at timestamp(6) without time zone,
    discarded_at timestamp(6) without time zone,
    finished_at timestamp(6) without time zone
);


--
-- Name: good_job_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.good_job_executions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    active_job_id uuid NOT NULL,
    job_class text,
    queue_name text,
    serialized_params jsonb,
    scheduled_at timestamp(6) without time zone,
    finished_at timestamp(6) without time zone,
    error text,
    error_event smallint
);


--
-- Name: good_job_processes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.good_job_processes (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    state jsonb
);


--
-- Name: good_job_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.good_job_settings (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    key text,
    value jsonb
);


--
-- Name: good_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.good_jobs (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    queue_name text,
    priority integer,
    serialized_params jsonb,
    scheduled_at timestamp(6) without time zone,
    performed_at timestamp(6) without time zone,
    finished_at timestamp(6) without time zone,
    error text,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    active_job_id uuid,
    concurrency_key text,
    cron_key text,
    retried_good_job_id uuid,
    cron_at timestamp(6) without time zone,
    batch_id uuid,
    batch_callback_id uuid,
    is_discrete boolean,
    executions_count integer,
    job_class text,
    error_event smallint
);


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    notifiable_type character varying NOT NULL,
    notified_at timestamp without time zone NOT NULL,
    read_at timestamp without time zone NOT NULL,
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
    comment text NOT NULL,
    stamps_count integer DEFAULT 0 NOT NULL,
    published_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    discarded_at timestamp(6) without time zone
);


--
-- Name: profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profileable_type character varying NOT NULL,
    atname public.citext NOT NULL,
    name character varying DEFAULT ''::character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    avatar_url character varying DEFAULT ''::character varying NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    discarded_at timestamp(6) without time zone
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: stamp_notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stamp_notifications (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    notification_id uuid NOT NULL,
    profile_id uuid NOT NULL,
    post_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: stamps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stamps (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    post_id uuid NOT NULL,
    stamped_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    profile_id uuid NOT NULL,
    email character varying NOT NULL,
    password_digest character varying NOT NULL,
    locale character varying NOT NULL,
    sign_in_count integer DEFAULT 0 NOT NULL,
    current_signed_in_at timestamp without time zone,
    last_signed_in_at timestamp without time zone,
    signed_up_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    time_zone character varying DEFAULT 'UTC'::character varying NOT NULL
);


--
-- Name: actors actors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.actors
    ADD CONSTRAINT actors_pkey PRIMARY KEY (id);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: email_confirmations email_confirmations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_confirmations
    ADD CONSTRAINT email_confirmations_pkey PRIMARY KEY (id);


--
-- Name: follow_notifications follow_notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follow_notifications
    ADD CONSTRAINT follow_notifications_pkey PRIMARY KEY (id);


--
-- Name: follows follows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT follows_pkey PRIMARY KEY (id);


--
-- Name: good_job_batches good_job_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.good_job_batches
    ADD CONSTRAINT good_job_batches_pkey PRIMARY KEY (id);


--
-- Name: good_job_executions good_job_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.good_job_executions
    ADD CONSTRAINT good_job_executions_pkey PRIMARY KEY (id);


--
-- Name: good_job_processes good_job_processes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.good_job_processes
    ADD CONSTRAINT good_job_processes_pkey PRIMARY KEY (id);


--
-- Name: good_job_settings good_job_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.good_job_settings
    ADD CONSTRAINT good_job_settings_pkey PRIMARY KEY (id);


--
-- Name: good_jobs good_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.good_jobs
    ADD CONSTRAINT good_jobs_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


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
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: stamp_notifications stamp_notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamp_notifications
    ADD CONSTRAINT stamp_notifications_pkey PRIMARY KEY (id);


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
-- Name: index_actors_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_actors_on_profile_id ON public.actors USING btree (profile_id);


--
-- Name: index_actors_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_actors_on_user_id ON public.actors USING btree (user_id);


--
-- Name: index_actors_on_user_id_and_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_actors_on_user_id_and_profile_id ON public.actors USING btree (user_id, profile_id);


--
-- Name: index_email_confirmations_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_email_confirmations_on_created_at ON public.email_confirmations USING btree (created_at);


--
-- Name: index_email_confirmations_on_email_and_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_email_confirmations_on_email_and_code ON public.email_confirmations USING btree (email, code);


--
-- Name: index_follow_notifications_on_notification_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_follow_notifications_on_notification_id ON public.follow_notifications USING btree (notification_id);


--
-- Name: index_follow_notifications_on_source_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_follow_notifications_on_source_profile_id ON public.follow_notifications USING btree (source_profile_id);


--
-- Name: index_follow_notifications_on_src_profile_id_and_tgt_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_follow_notifications_on_src_profile_id_and_tgt_profile_id ON public.follow_notifications USING btree (source_profile_id, target_profile_id);


--
-- Name: index_follow_notifications_on_target_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_follow_notifications_on_target_profile_id ON public.follow_notifications USING btree (target_profile_id);


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
-- Name: index_good_job_executions_on_active_job_id_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_job_executions_on_active_job_id_and_created_at ON public.good_job_executions USING btree (active_job_id, created_at);


--
-- Name: index_good_job_settings_on_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_good_job_settings_on_key ON public.good_job_settings USING btree (key);


--
-- Name: index_good_jobs_jobs_on_finished_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_jobs_on_finished_at ON public.good_jobs USING btree (finished_at) WHERE ((retried_good_job_id IS NULL) AND (finished_at IS NOT NULL));


--
-- Name: index_good_jobs_jobs_on_priority_created_at_when_unfinished; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_jobs_on_priority_created_at_when_unfinished ON public.good_jobs USING btree (priority DESC NULLS LAST, created_at) WHERE (finished_at IS NULL);


--
-- Name: index_good_jobs_on_active_job_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_active_job_id ON public.good_jobs USING btree (active_job_id);


--
-- Name: index_good_jobs_on_active_job_id_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_active_job_id_and_created_at ON public.good_jobs USING btree (active_job_id, created_at);


--
-- Name: index_good_jobs_on_batch_callback_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_batch_callback_id ON public.good_jobs USING btree (batch_callback_id) WHERE (batch_callback_id IS NOT NULL);


--
-- Name: index_good_jobs_on_batch_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_batch_id ON public.good_jobs USING btree (batch_id) WHERE (batch_id IS NOT NULL);


--
-- Name: index_good_jobs_on_concurrency_key_when_unfinished; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_concurrency_key_when_unfinished ON public.good_jobs USING btree (concurrency_key) WHERE (finished_at IS NULL);


--
-- Name: index_good_jobs_on_cron_key_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_cron_key_and_created_at ON public.good_jobs USING btree (cron_key, created_at);


--
-- Name: index_good_jobs_on_cron_key_and_cron_at; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_good_jobs_on_cron_key_and_cron_at ON public.good_jobs USING btree (cron_key, cron_at);


--
-- Name: index_good_jobs_on_queue_name_and_scheduled_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_queue_name_and_scheduled_at ON public.good_jobs USING btree (queue_name, scheduled_at) WHERE (finished_at IS NULL);


--
-- Name: index_good_jobs_on_scheduled_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_good_jobs_on_scheduled_at ON public.good_jobs USING btree (scheduled_at) WHERE (finished_at IS NULL);


--
-- Name: index_notifications_on_notified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notifications_on_notified_at ON public.notifications USING btree (notified_at);


--
-- Name: index_notifications_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notifications_on_profile_id ON public.notifications USING btree (profile_id);


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
-- Name: index_oauth_applications_on_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_applications_on_name ON public.oauth_applications USING btree (name);


--
-- Name: index_oauth_applications_on_uid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_applications_on_uid ON public.oauth_applications USING btree (uid);


--
-- Name: index_posts_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_posts_on_discarded_at ON public.posts USING btree (discarded_at);


--
-- Name: index_posts_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_posts_on_profile_id ON public.posts USING btree (profile_id);


--
-- Name: index_profiles_on_atname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_profiles_on_atname ON public.profiles USING btree (atname);


--
-- Name: index_profiles_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_profiles_on_discarded_at ON public.profiles USING btree (discarded_at);


--
-- Name: index_stamp_notifications_on_notification_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamp_notifications_on_notification_id ON public.stamp_notifications USING btree (notification_id);


--
-- Name: index_stamp_notifications_on_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamp_notifications_on_post_id ON public.stamp_notifications USING btree (post_id);


--
-- Name: index_stamp_notifications_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamp_notifications_on_profile_id ON public.stamp_notifications USING btree (profile_id);


--
-- Name: index_stamp_notifications_on_profile_id_and_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_stamp_notifications_on_profile_id_and_post_id ON public.stamp_notifications USING btree (profile_id, post_id);


--
-- Name: index_stamps_on_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamps_on_post_id ON public.stamps USING btree (post_id);


--
-- Name: index_stamps_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stamps_on_profile_id ON public.stamps USING btree (profile_id);


--
-- Name: index_stamps_on_profile_id_and_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_stamps_on_profile_id_and_post_id ON public.stamps USING btree (profile_id, post_id);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: index_users_on_profile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_profile_id ON public.users USING btree (profile_id);


--
-- Name: stamps fk_rails_27da15755d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamps
    ADD CONSTRAINT fk_rails_27da15755d FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: follow_notifications fk_rails_32749f329d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follow_notifications
    ADD CONSTRAINT fk_rails_32749f329d FOREIGN KEY (target_profile_id) REFERENCES public.profiles(id);


--
-- Name: oauth_access_grants fk_rails_330c32d8d9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_grants
    ADD CONSTRAINT fk_rails_330c32d8d9 FOREIGN KEY (resource_owner_id) REFERENCES public.profiles(id);


--
-- Name: actors fk_rails_477d25671f; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.actors
    ADD CONSTRAINT fk_rails_477d25671f FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: stamp_notifications fk_rails_4c86a04e8f; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamp_notifications
    ADD CONSTRAINT fk_rails_4c86a04e8f FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: actors fk_rails_541f6b56e7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.actors
    ADD CONSTRAINT fk_rails_541f6b56e7 FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: follows fk_rails_5e22b9865a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_5e22b9865a FOREIGN KEY (source_profile_id) REFERENCES public.profiles(id);


--
-- Name: stamp_notifications fk_rails_63530a6a31; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamp_notifications
    ADD CONSTRAINT fk_rails_63530a6a31 FOREIGN KEY (notification_id) REFERENCES public.notifications(id);


--
-- Name: stamps fk_rails_6933333714; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamps
    ADD CONSTRAINT fk_rails_6933333714 FOREIGN KEY (post_id) REFERENCES public.posts(id);


--
-- Name: oauth_access_tokens fk_rails_732cb83ab7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_tokens
    ADD CONSTRAINT fk_rails_732cb83ab7 FOREIGN KEY (application_id) REFERENCES public.oauth_applications(id);


--
-- Name: stamp_notifications fk_rails_a395841b48; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stamp_notifications
    ADD CONSTRAINT fk_rails_a395841b48 FOREIGN KEY (post_id) REFERENCES public.posts(id);


--
-- Name: users fk_rails_a8794354f0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT fk_rails_a8794354f0 FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: follows fk_rails_abf061b756; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_rails_abf061b756 FOREIGN KEY (target_profile_id) REFERENCES public.profiles(id);


--
-- Name: follow_notifications fk_rails_b2df4597e6; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follow_notifications
    ADD CONSTRAINT fk_rails_b2df4597e6 FOREIGN KEY (notification_id) REFERENCES public.notifications(id);


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
-- Name: notifications fk_rails_e251e4b22b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT fk_rails_e251e4b22b FOREIGN KEY (profile_id) REFERENCES public.profiles(id);


--
-- Name: oauth_access_tokens fk_rails_ee63f25419; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_access_tokens
    ADD CONSTRAINT fk_rails_ee63f25419 FOREIGN KEY (resource_owner_id) REFERENCES public.actors(id);


--
-- Name: follow_notifications fk_rails_f0e87a56bd; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.follow_notifications
    ADD CONSTRAINT fk_rails_f0e87a56bd FOREIGN KEY (source_profile_id) REFERENCES public.profiles(id);


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
('20221119004702'),
('20230830155958'),
('20231015050126'),
('20231025165318'),
('20231028081207');


