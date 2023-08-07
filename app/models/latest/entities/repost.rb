# typed: strict
# frozen_string_literal: true

class Latest::Entities::Repost < Latest::Entities::Base
  sig { params(repost: Repost, viewer: Profile).void }
  def initialize(repost:, viewer:)
    @repost = repost
    @viewer = viewer
  end

  sig { returns(Latest::Entities::Post) }
  def target_post
    Latest::Entities::Post.new(post: repost.target_post!, viewer:)
  end

  sig { returns(Latest::Entities::Profile) }
  def target_profile
    Latest::Entities::Profile.new(profile: repost.target_profile!)
  end

  sig { returns(Latest::Entities::Post) }
  def original_post
    Latest::Entities::Post.new(post: repost.original_post!, viewer:)
  end

  sig { returns(Latest::Entities::Profile) }
  def original_profile
    Latest::Entities::Profile.new(profile: repost.original_profile!)
  end

  sig { returns(Repost) }
  attr_reader :repost
  private :repost

  sig { returns(Profile) }
  attr_reader :viewer
  private :viewer
end
