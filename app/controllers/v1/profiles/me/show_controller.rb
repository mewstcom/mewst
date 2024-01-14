# typed: true
# frozen_string_literal: true

class V1::Profiles::Me::ShowController < V1::ApplicationController
  include PublicAuthenticatable

  def call
    profile_resource = V1::ProfileResource.new(viewer: current_viewer!, profile: current_viewer!.profile.not_nil!)

    render(
      json: {
        profile: V1::ProfileSerializer.new(profile_resource).to_h
      }
    )
  end
end
