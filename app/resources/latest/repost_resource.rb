# typed: strict
# frozen_string_literal: true

class Latest::RepostResource < Latest::ApplicationResource
  sig { params(repost: Repost, viewer: Profile).void }
  def initialize(repost:, viewer:)
    @repost = repost
    @viewer = viewer
  end

  sig { returns(Latest::Entities::Profile) }
  def profile
    Latest::Entities::Profile.new(profile: repost.profile!)
  end

  sig { returns(Latest::Entities::Post) }
  def original_post
    Latest::Entities::Post.new(post: repost.original_post, viewer:)
  end

  sig { returns(Repost) }
  attr_reader :repost
  private :repost

  sig { returns(Profile) }
  attr_reader :viewer
  private :viewer
end
