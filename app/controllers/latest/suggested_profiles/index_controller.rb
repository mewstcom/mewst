# typed: true
# frozen_string_literal: true

class Latest::SuggestedProfiles::IndexController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    records = current_viewer.not_nil!.suggested_followees.kept.merge(SuggestedFollow.not_checked)
    result = Paginator.new(records:).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    profile_resources = result.records.map { |profile| Latest::ProfileResource.new(profile:, viewer: current_viewer) }

    render(
      json: {
        profiles: Latest::ProfileSerializer.new(profile_resources).to_h,
        page_info: Latest::PageInfoSerializer.new(result.page_info).to_h
      }
    )
  end
end
