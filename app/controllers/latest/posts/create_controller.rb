# typed: true
# frozen_string_literal: true

class Latest::Posts::CreateController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    form = Latest::PostForm.new(
      profile: current_profile.not_nil!,
      comment: params[:comment]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    input = CreatePostService::Input.from_latest_form(form:)
    result = ActiveRecord::Base.transaction do
      CreatePostService.new.call(input:)
    end

    resource = Latest::PostResource.new(post: result.post, viewer: current_profile.not_nil!)
    render(
      json: Latest::PostSerializer.new(resource),
      status: :created
    )
  end
end
