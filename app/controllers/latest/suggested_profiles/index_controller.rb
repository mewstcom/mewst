# typed: true
# frozen_string_literal: true

class Latest::SuggestedProfiles::IndexController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    profiles = current_viewer.not_nil!.suggested_followees.kept.merge(SuggestedFollow.not_checked).order(id: :desc).limit(30)
    profile_resources = profiles.map { |profile| Latest::ProfileResource.new(profile:, viewer: current_viewer) }

    render(
      json: {
        profiles: Latest::ProfileSerializer.new(profile_resources).to_h
      }
    )
  end
end
