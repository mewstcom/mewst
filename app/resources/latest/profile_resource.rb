# typed: strict
# frozen_string_literal: true

class Latest::ProfileResource < Latest::ApplicationResource
  delegate :atname, :avatar_url, :description, :name, to: :profile

  sig { params(profile: Profile, viewer: T.nilable(Actor)).void }
  def initialize(profile:, viewer:)
    @profile = profile
    @viewer = viewer
  end

  sig { returns(T::Boolean) }
  def viewer_has_followed
    return false if viewer.nil?

    viewer.not_nil!.following?(target_profile: profile)
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  sig { returns(T.nilable(Actor)) }
  attr_reader :viewer
  private :viewer
end
