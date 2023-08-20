# typed: true
# frozen_string_literal: true

class Latest::Follows::CreateController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    form = Latest::FollowForm.new(
      viewer: current_profile.not_nil!,
      target_atname: params[:atname]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FollowFormErrorResource, errors: form.errors)
    end

    result = ActiveRecord::Base.transaction do
      input = FollowProfileService::Input.from_latest_form(form:)
      FollowProfileService.new.call(input:)
    end

    resource = Latest::ProfileResource.new(profile: result.target_profile, viewer: current_profile.not_nil!)
    render(
      json: Latest::ProfileSerializer.new(resource),
      status: :created
    )
  end
end
