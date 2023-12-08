# typed: strict
# frozen_string_literal: true

module Mewst::Test::ResourceHelpers
  extend T::Sig

  sig { params(email_confirmation: EmailConfirmation).returns(T::Hash[Symbol, T.untyped]) }
  def build_email_confirmation_resource(email_confirmation:)
    {
      id: email_confirmation.id,
      email: email_confirmation.email,
      event: email_confirmation.event,
      succeeded_at: email_confirmation.succeeded_at&.iso8601
    }
  end

  sig { params(profile: Profile, viewer_has_followed: T::Boolean).returns(T::Hash[Symbol, T.untyped]) }
  def build_profile_resource(profile:, viewer_has_followed:)
    {
      id: profile.id,
      atname: profile.atname,
      name: profile.name,
      description: profile.description,
      avatar_url: profile.avatar_url,
      viewer_has_followed:
    }
  end

  sig { params(post: Post, viewer_has_followed: T::Boolean, viewer_has_stamped: T::Boolean).returns(T::Hash[Symbol, T.untyped]) }
  def build_post_resource(post:, viewer_has_followed:, viewer_has_stamped:)
    {
      profile: build_profile_resource(profile: post.profile.not_nil!, viewer_has_followed:),
      id: post.id,
      content: post.content,
      published_at: post.published_at.iso8601,
      stamps_count: post.stamps_count,
      viewer_has_stamped:,
      via: build_via_resource(oauth_application: post.oauth_application.not_nil!)
    }
  end

  sig { params(user: User).returns(T::Hash[Symbol, T.untyped]) }
  def build_user_resource(user:)
    {
      id: user.id,
      locale: user.locale,
      time_zone: user.time_zone
    }
  end

  sig { params(oauth_application: OauthApplication).returns(T::Hash[Symbol, T.untyped]) }
  def build_via_resource(oauth_application:)
    {
      name: oauth_application.name
    }
  end
end
