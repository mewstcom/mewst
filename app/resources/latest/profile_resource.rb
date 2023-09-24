# typed: strict
# frozen_string_literal: true

class Latest::ProfileResource < Latest::ApplicationResource
  delegate :atname, :avatar_url, :description, :name, to: :profile

  sig { params(viewer: T.nilable(Actor), profile: Profile).void }
  def initialize(profile:, viewer:)
    @viewer = viewer
    @profile = profile
  end

  sig { returns(T::Boolean) }
  def viewer_has_followed
    return false if viewer.nil?

    viewer.not_nil!.following?(target_profile: profile)
  end

  sig { returns(T.nilable(Actor)) }
  attr_reader :viewer
  private :viewer

  sig { returns(Profile) }
  attr_reader :profile
  private :profile
end
