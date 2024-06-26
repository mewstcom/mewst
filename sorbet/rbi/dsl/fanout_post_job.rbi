# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `FanoutPostJob`.
# Please instead update this file by running `bin/tapioca dsl FanoutPostJob`.


class FanoutPostJob
  class << self
    sig do
      params(
        post_id: ::String,
        block: T.nilable(T.proc.params(job: FanoutPostJob).void)
      ).returns(T.any(FanoutPostJob, FalseClass))
    end
    def perform_later(post_id:, &block); end

    sig { params(post_id: ::String).void }
    def perform_now(post_id:); end
  end
end
