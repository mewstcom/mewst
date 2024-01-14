# typed: true
# frozen_string_literal: true

class V1::SuggestedProfiles::IndexController < V1::ApplicationController
  include ControllerConcerns::PublicAuthenticatable

  def call
    profiles = current_viewer.not_nil!.suggested_followees.kept.merge(SuggestedFollow.not_checked).order(created_at: :desc).limit(30)
    profile_resources = profiles.map { |profile| V1::ProfileResource.new(profile:, viewer: current_viewer) }

    render(
      json: {
        profiles: V1::ProfileSerializer.new(profile_resources).to_h
      }
    )
  end
end
