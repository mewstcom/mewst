# typed: strict
# frozen_string_literal: true

class V1::ProfileResource < V1::ApplicationResource
  delegate :id, :atname, :name, :description, :avatar_url, to: :profile

  sig { params(viewer: T.nilable(ActorRecord), profile: ProfileRecord).void }
  def initialize(viewer:, profile:)
    @viewer = viewer
    @profile = profile
  end

  sig { returns(T::Boolean) }
  def viewer_has_followed
    return false if viewer.nil?

    viewer.not_nil!.following?(target_profile: profile)
  end

  sig { returns(T.nilable(ActorRecord)) }
  attr_reader :viewer
  private :viewer

  sig { returns(ProfileRecord) }
  attr_reader :profile
  private :profile
end
