# typed: strict
# frozen_string_literal: true

class Profile::Timeline::Home
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(post: Post).returns(T.self_type) }
  def add_post(post:)
    T.cast(Mewst::Redis.client, Redis).zadd(profile.timeline_key, post.timeline_item_score, post.id)

    self
  end

  sig { params(post: Post).returns(T.self_type) }
  def remove_post(post:)
    T.cast(Mewst::Redis.client, Redis).zrem(profile.timeline_key, post.id)

    self
  end

  private

  sig { returns(Profile) }
  attr_reader :profile
end
