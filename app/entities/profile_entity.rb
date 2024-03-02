# typed: strict
# frozen_string_literal: true

class ProfileEntity < ApplicationEntity
  sig { returns(String) }
  attr_reader :atname

  sig { returns(String) }
  attr_reader :name

  sig { returns(String) }
  attr_reader :description

  sig { returns(String) }
  attr_reader :avatar_url

  sig { returns(T::Boolean) }
  attr_reader :viewer_has_followed

  sig { params(atname: String, name: String, description: String, avatar_url: String, viewer_has_followed: T::Boolean).void }
  def initialize(atname:, name:, description:, avatar_url:, viewer_has_followed:)
    @atname = atname
    @name = name
    @description = description
    @avatar_url = avatar_url
    @viewer_has_followed = viewer_has_followed
  end

  sig { params(profile: Profile, follow_checker: FollowChecker).returns(T.self_type) }
  def self.from_model(profile:, follow_checker:)
    new(
      atname: profile.atname,
      name: profile.name,
      description: profile.description,
      avatar_url: profile.avatar_url,
      viewer_has_followed: follow_checker.followed?(target_profile: profile)
    )
  end
end
