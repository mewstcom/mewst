# typed: true
# frozen_string_literal: true

class Latest::Stamps::DestroyController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    form = Latest::StampForm.new(
      profile: current_profile.not_nil!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::StampFormErrorResource, errors: form.errors)
    end

    result = ActiveRecord::Base.transaction do
      input = DeleteStampService::Input.from_latest_form(form:)
      DeleteStampService.new.call(input:)
    end

    resource = Latest::PostResource.new(post: result.post, viewer: current_profile.not_nil!)
    render(
      json: Latest::PostSerializer.new(resource),
      status: :created
    )
  end
end
