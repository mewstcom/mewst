# typed: true
# frozen_string_literal: true

class Latest::CommentPosts::CreateController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    form = Latest::CommentPostForm.new(
      comment: params[:comment]
    )
    form.profile = current_profile!

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    input = CreateCommentPostService::Input.from_latest_form(form:)
    result = ActiveRecord::Base.transaction do
      CreateCommentPostService.new.call(input:)
    end

    resource = Latest::PostResource.new(post: result.post, viewer: current_profile!)
    render(
      json: Latest::PostSerializer.new(resource),
      status: :created
    )
  end
end
